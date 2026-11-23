#!/usr/bin/env python3
"""Exercise HTTP -> gRPC -> PostgreSQL using disposable databases and processes.

Requires `make build`, Python 3 and PostgreSQL 14+ server tools on PATH (or
PG_BINDIR). All ports are selected dynamically; no existing database is touched.
"""
import base64
from concurrent.futures import ThreadPoolExecutor
import hashlib
import hmac
import json
import os
from pathlib import Path
import re
import shutil
import socket
import subprocess
import tempfile
import time
from threading import Barrier
import urllib.error
import urllib.request
from urllib.parse import urlencode
import uuid

ROOT = Path(__file__).resolve().parents[1]
checks = []
requests = []


def port():
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        return sock.getsockname()[1]


def pg_tool(name):
    location = os.environ.get("PG_BINDIR")
    result = str(Path(location) / name) if location else shutil.which(name)
    if not result or not Path(result).is_file():
        raise RuntimeError(f"PostgreSQL tool {name} missing; set PG_BINDIR")
    return result


def stop(process):
    if process and process.poll() is None:
        process.terminate()
        try:
            process.wait(timeout=10)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait(timeout=5)


def run():
    for binary in ("auth", "shop"):
        if not (ROOT / "bin" / binary).is_file():
            raise RuntimeError("Run make build first")
    with tempfile.TemporaryDirectory(prefix="be4-integration-") as directory:
        temp = Path(directory)
        db_port, auth_port, http_port = port(), port(), port()
        data = temp / "pg"
        pg_log = temp / "postgres.log"
        auth_log = temp / "auth.log"
        shop_log = temp / "shop.log"
        auth = shop = None
        pg_started = False
        secret = os.urandom(32).hex()
        env = dict(os.environ, AUTH_DB_DSN=f"postgres://postgres@127.0.0.1:{db_port}/auth_db?sslmode=disable",
                   AUTH_GRPC_ADDR=f"127.0.0.1:{auth_port}", AUTH_JWT_SECRET=secret, AUTH_TOKEN_TTL="1h",
                   AUTH_JWT_ISSUER="authservice", AUTH_JWT_AUDIENCE="shopapi",
                   PORT=str(http_port), DB_HOST="127.0.0.1", DB_PORT=str(db_port), DB_USER="postgres",
                   DB_PASSWORD="", DB_NAME="shopapi", DB_SSLMODE="disable", DB_MIGRATE="true", GIN_MODE="release")
        pg_args = ["-h", "127.0.0.1", "-p", str(db_port), "-U", "postgres"]
        pg_env = {key: value for key, value in os.environ.items() if not key.startswith("PG")}

        def sql(database, query):
            return subprocess.check_output([pg_tool("psql"), *pg_args, "-d", database, "-XAt",
                                            "-v", "ON_ERROR_STOP=1", "-c", query], text=True, env=pg_env).strip()

        def call(method, path, payload=None, token=None):
            body = json.dumps(payload).encode() if payload is not None else None
            headers = {"Content-Type": "application/json"}
            if token:
                headers["Authorization"] = "Bearer " + token
            req = urllib.request.Request(f"http://127.0.0.1:{http_port}{path}", body, headers, method=method)
            try:
                response = urllib.request.urlopen(req, timeout=6)
            except urllib.error.HTTPError as error:
                response = error
            with response:
                raw = response.read()
                requests.append({"method": method, "path": path, "status": response.status})
                try:
                    result = json.loads(raw) if raw else None
                except (ValueError, UnicodeError):
                    result = raw
                return response.status, result

        def expect(code, method, path, payload=None, token=None):
            actual, body = call(method, path, payload, token)
            if actual != code:
                raise AssertionError(f"{method} {path}: expected {code}, got {actual}: {body}")
            return body

        def ready():
            deadline = time.monotonic() + 20
            while time.monotonic() < deadline:
                if auth.poll() is not None or shop.poll() is not None:
                    raise RuntimeError("Application exited during startup")
                try:
                    if call("POST", "/auth", {"email": "missing@example.com", "password": "missing"})[0] == 401:
                        return
                except (OSError, urllib.error.URLError):
                    pass
                time.sleep(0.1)
            raise RuntimeError("Applications did not become ready")

        try:
            subprocess.run([pg_tool("initdb"), "-D", str(data), "-A", "trust", "-U", "postgres", "--no-locale", "--encoding=UTF8"],
                           check=True, stdout=subprocess.DEVNULL, env=pg_env)
            # Disable Unix sockets to avoid platform path-length limits.
            subprocess.run([pg_tool("pg_ctl"), "-D", str(data), "-l", str(pg_log), "-o",
                            f"-p {db_port} -h 127.0.0.1 -k ''", "-w", "start"], check=True,
                           stdout=subprocess.DEVNULL, env=pg_env)
            pg_started = True
            for database in ("shopapi", "auth_db"):
                subprocess.run([pg_tool("createdb"), *pg_args, database], check=True, env=pg_env)
            with auth_log.open("w") as auth_output, shop_log.open("w") as shop_output:
                auth = subprocess.Popen([ROOT / "bin/auth"], env=env, stdout=auth_output, stderr=subprocess.STDOUT)
                shop = subprocess.Popen([ROOT / "bin/shop"], env=env, stdout=shop_output, stderr=subprocess.STDOUT)
                ready()
                user = {"email": "ivan@example.com", "first_name": "Ivan", "last_name": "Ivanov",
                        "phone": "+79990000000", "password": "initial-password"}
                token = expect(201, "POST", "/register", user)["token"]
                expect(409, "POST", "/register", user)
                expect(400, "POST", "/register", dict(user, email="bad-email"))
                expect(400, "POST", "/register", dict(user, email="long@example.com", password="x" * 73))
                expect(201, "POST", "/register", dict(user, email="second@example.com"))
                hashes = sql("auth_db", "SELECT password_hash FROM users ORDER BY email").splitlines()
                assert len(hashes) == 2 and hashes[0] != hashes[1] and all(h.startswith("$2") for h in hashes)
                assert sql("shopapi", "SELECT to_regclass('public.users') IS NULL") == "t"
                assert sql("auth_db", "SELECT to_regclass('public.products') IS NULL") == "t"
                credentials = {"email": user["email"], "password": user["password"]}
                expect(200, "POST", "/auth", credentials)
                expect(401, "POST", "/auth", dict(credentials, password="wrong"))
                for resource in ("clients", "suppliers", "products", "images/00000000-0000-0000-0000-000000000001"):
                    expect(401, "GET", "/api/v1/" + resource)
                expect(200, "GET", "/api/v1/products", token=token)
                expect(401, "GET", "/api/v1/products", token=token + "tampered")

                def encode(value):
                    return base64.urlsafe_b64encode(json.dumps(value).encode()).rstrip(b"=")
                claims = json.loads(base64.urlsafe_b64decode(token.split(".")[1] + "=="))
                claims["exp"] = int(time.time()) - 60
                unsigned = encode({"alg": "HS256", "typ": "JWT"}) + b"." + encode(claims)
                signature = base64.urlsafe_b64encode(hmac.new(secret.encode(), unsigned, hashlib.sha256).digest()).rstrip(b"=")
                expired = (unsigned + b"." + signature).decode()
                expect(401, "GET", "/api/v1/products", token=expired)
                print("PASS registration, login, salted hashes, database separation, invalid and expired JWT", flush=True)
                # Collect independent failures so one defect does not hide other regressions.
                def check(name, condition, details=None):
                    result = {"name": name, "passed": bool(condition)}
                    if details is not None:
                        result["details"] = details
                    checks.append(result)
                    print(("PASS " if condition else "FAIL ") + name + (": " + str(details) if not condition else ""), flush=True)

                swagger = json.loads((ROOT / "ShopAPI/docs/swagger.json").read_text())
                for path, methods in swagger["paths"].items():
                    if not (path.startswith("/api/v1/") or path == "/change-password"):
                        continue
                    concrete = re.sub(r"\{[^}]+\}", "00000000-0000-0000-0000-000000000001", path)
                    for method in methods:
                        for credential in (None, "invalid.jwt.token"):
                            code, _ = call(method.upper(), concrete, {}, credential)
                            check(f"guard {method.upper()} {path} {'missing' if credential is None else 'forged'}", code == 401, code)

                for field, value in [("email", "bad"), ("first_name", " "), ("last_name", ""), ("phone", ""),
                                     ("password", ""), ("password", "x" * 73), ("password", "я" * 37),
                                     ("first_name", "x" * 101), ("last_name", "x" * 101), ("phone", "x" * 21)]:
                    registration_data = dict(user, email=uuid.uuid4().hex + "@example.com")
                    registration_data[field] = value
                    code, _ = call("POST", "/register", registration_data)
                    check(f"register validates {field} length={len(value)}", code == 400, code)
                # PostgreSQL cannot store U+0000; verify validation handles it as bad input.
                code, _ = call("POST", "/register", dict(user, email="nul@example.com", first_name="Ivan\x00Test"))
                check("register rejects NUL in name as client error", code == 400, code)
                code, raw = call("POST", "/register", dict(user, email="unicode@example.com", first_name="Иван", last_name="Иванов", password="я" * 36))
                check("register accepts Unicode names and 72-byte password", code == 201, code)
                code, _ = call("POST", "/auth", {"email": "unicode@example.com", "password": "я" * 36})
                check("login with Unicode password", code == 200, code)

                # Boundary / adversarial JWT cases exercise the entire HTTP->gRPC path.
                valid_claims = json.loads(base64.urlsafe_b64decode(token.split(".")[1] + "=="))
                def signed(claims, algorithm="HS256", key=secret):
                    encoded = lambda value: base64.urlsafe_b64encode(json.dumps(value).encode()).rstrip(b"=")
                    unsigned = encoded({"alg": algorithm, "typ": "JWT"}) + b"." + encoded(claims)
                    digest = hashlib.sha512 if algorithm == "HS512" else hashlib.sha256
                    signature = b"" if algorithm == "none" else base64.urlsafe_b64encode(hmac.new(key.encode(), unsigned, digest).digest()).rstrip(b"=")
                    return (unsigned + b"." + signature).decode()
                mutations = {"expired": {"exp": int(time.time())-1}, "future nbf": {"nbf": int(time.time())+600},
                             "future iat": {"iat": int(time.time())+600}, "wrong issuer": {"iss": "wrong"},
                             "wrong audience": {"aud": "wrong"}, "empty subject": {"sub": ""}, "invalid subject": {"sub": "wrong"},
                             "empty email": {"email": ""}}
                for name, patch in mutations.items():
                    code, _ = call("GET", "/api/v1/products", token=signed(dict(valid_claims, **patch)))
                    check("JWT " + name, code == 401, code)
                no_exp = dict(valid_claims)
                del no_exp["exp"]
                for name, candidate in [("missing exp", signed(no_exp)), ("none", signed(valid_claims, "none")),
                                         ("HS512", signed(valid_claims, "HS512")), ("wrong key", signed(valid_claims, key="wrong-secret"))]:
                    check("JWT " + name, call("GET", "/api/v1/products", token=candidate)[0] == 401)

                # Concurrent registration must produce one row and one successful response.
                parallel_user = dict(user, email="race-register@example.com")
                barrier = Barrier(8)
                def register_parallel(_):
                    barrier.wait()
                    return call("POST", "/register", parallel_user)[0]
                with ThreadPoolExecutor(max_workers=8) as pool:
                    results = list(pool.map(register_parallel, range(8)))
                check("concurrent registration uniqueness", results.count(201)==1 and results.count(409)==7, results)

                # Complete store CRUD including content, links and persistence after PATCH.
                address = sql("shopapi", "INSERT INTO addresses (country,city,street) VALUES ('RU','OldCity','OldStreet') RETURNING id").splitlines()[0]
                category = sql("shopapi", "INSERT INTO categories (name) VALUES ('ReviewCategory') RETURNING id").splitlines()[0]
                client_data = {"client_name":"Review", "client_surname":"Client", "birthday":"1990-01-02", "gender":"M", "address_id":address}
                code, client = call("POST", "/api/v1/clients", client_data, token)
                check("client create", code==201, code)
                cid = client["id"]
                code, fetched = call("GET", "/api/v1/clients/"+cid, token=token)
                check("client read", code==200 and fetched["client_name"]=="Review")
                code, found = call("GET", "/api/v1/clients/search?"+urlencode({"name":"Review","surname":"Client"}), token=token)
                check("client search", code==200 and any(c["id"]==cid for c in found))
                code, listed = call("GET", "/api/v1/clients?limit=1&offset=0", token=token)
                check("client list pagination", code==200 and len(listed)==1)
                new_address = {"address":{"country":"RU","city":"NewCity","street":"NewStreet"}}
                code, _ = call("PATCH", "/api/v1/clients/"+cid+"/address", new_address, token)
                check("client PATCH returns 204", code==204, code)
                code, fetched = call("GET", "/api/v1/clients/"+cid, token=token)
                check("client PATCH actually changes address", fetched["address"]["city"]=="NewCity", fetched["address"])
                supplier_data = {"name":"ReviewSupplier","address_id":address,"phone_number":"+79990000001"}
                code, supplier = call("POST", "/api/v1/suppliers", supplier_data, token)
                check("supplier create", code==201, code)
                sid = supplier["id"]
                code, listed = call("GET", "/api/v1/suppliers", token=token)
                check("supplier list", code==200 and any(s["id"]==sid for s in listed))
                code, _ = call("PATCH", "/api/v1/suppliers/"+sid+"/address", new_address, token)
                check("supplier PATCH returns 204", code==204, code)
                code, fetched = call("GET", "/api/v1/suppliers/"+sid, token=token)
                check("supplier PATCH actually changes address", code==200 and fetched["address"]["city"]=="NewCity", fetched["address"])
                product_data = {"name":"ReviewProduct","category_id":category,"supplier_id":sid,"price":100,"available_stock":100}
                code, product = call("POST", "/api/v1/products", product_data, token)
                check("product create", code==201, code)
                pid = product["id"]
                code, products = call("GET", "/api/v1/products", token=token)
                check("product list", code==200 and any(p["id"]==pid for p in products))
                code, _ = call("PATCH", "/api/v1/products/"+pid+"/decrease", {"amount":2}, token)
                check("product decrease returns 204", code==204, code)
                code, fetched = call("GET", "/api/v1/products/"+pid, token=token)
                check("product decrease persisted", fetched["available_stock"]==98, fetched["available_stock"])
                check("product cannot go below zero", call("PATCH", "/api/v1/products/"+pid+"/decrease", {"amount":1000}, token)[0]==400)
                check("product decrease rejects zero", call("PATCH", "/api/v1/products/"+pid+"/decrease", {"amount":0}, token)[0]==400)
                code, raw = call("POST", "/api/v1/products", dict(product_data, price=0), token)
                check("product allows price zero per DTO gte=0 and SQL constraint", code==201, {"status":code,"body":raw})
                if code==201:
                    call("DELETE", "/api/v1/products/"+raw["id"], token=token)

                def multipart(method, path, content, product_id=None):
                    boundary = "review-"+uuid.uuid4().hex
                    pieces=[]
                    if product_id:
                        pieces.append(f'--{boundary}\r\nContent-Disposition: form-data; name="product_id"\r\n\r\n{product_id}\r\n'.encode())
                    pieces.extend([f'--{boundary}\r\nContent-Disposition: form-data; name="image"; filename="test.png"\r\nContent-Type: image/png\r\n\r\n'.encode(),content,f'\r\n--{boundary}--\r\n'.encode()])
                    req=urllib.request.Request(f"http://127.0.0.1:{http_port}"+path,b"".join(pieces),{"Content-Type":"multipart/form-data; boundary="+boundary,"Authorization":"Bearer "+token},method=method)
                    try: response=urllib.request.urlopen(req,timeout=6)
                    except urllib.error.HTTPError as error: response=error
                    with response:
                        raw=response.read()
                        requests.append({"method":method,"path":path,"status":response.status})
                        return response.status,json.loads(raw) if raw else None
                image1=b"\x89PNG\r\n\x1a\nreview-first-content"
                image2=b"\x89PNG\r\n\x1a\nreview-second-content"
                code, uploaded=multipart("POST","/api/v1/images",image1,pid)
                check("image upload",code==201,code)
                iid=uploaded["id"]
                code, contents=call("GET","/api/v1/images/"+iid,token=token)
                check("image download identical bytes",code==200 and contents==image1)
                code, contents=call("GET","/api/v1/images/product/"+pid,token=token)
                check("image linked to product",code==200 and contents==image1)
                code, _=multipart("PUT","/api/v1/images/"+iid,image2)
                check("image replace",code==204,code)
                code, contents=call("GET","/api/v1/images/product/"+pid,token=token)
                check("image replacement persisted",code==200 and contents==image2)
                code, _=call("DELETE","/api/v1/images/"+iid,token=token)
                check("image delete",code==204,code)
                check("deleted image returns 404",call("GET","/api/v1/images/"+iid,token=token)[0]==404)
                check("deleted image unlinked from product",call("GET","/api/v1/images/product/"+pid,token=token)[0]==404)

                for label, fields in [("missing", {k:v for k,v in product_data.items() if k != "price"}),
                                      ("null", dict(product_data, price=None)), ("negative", dict(product_data, price=-1))]:
                    code, _ = call("POST", "/api/v1/products", fields, token)
                    check("product price rejects " + label, code == 400, code)

                # 16 simultaneous successful decrements must each deduct one unit.
                barrier=Barrier(16)
                def decrease_parallel(_):
                    barrier.wait()
                    return call("PATCH","/api/v1/products/"+pid+"/decrease",{"amount":1},token)[0]
                with ThreadPoolExecutor(max_workers=16) as pool:
                    statuses=list(pool.map(decrease_parallel,range(16)))
                code, fetched=call("GET","/api/v1/products/"+pid,token=token)
                check("concurrent stock decrements are not lost",statuses == [204] * 16 and code == 200 and fetched["available_stock"] == 82,
                      {"successful_requests":statuses.count(204),"initial":98,"actual":fetched["available_stock"],"expected":98-statuses.count(204)})
                code, raw=call("DELETE","/api/v1/suppliers/"+sid,token=token)
                check("referenced supplier deletion gives conflict",code==409,{"status":code,"body":raw})
                for resource, object_id in [("products",pid),("suppliers",sid),("clients",cid)]:
                    check(resource+" deletion",call("DELETE","/api/v1/"+resource+"/"+object_id,token=token)[0]==204)
                    check(resource+" deleted GET",call("GET","/api/v1/"+resource+"/"+object_id,token=token)[0]==404)
                    check(resource+" repeat deletion",call("DELETE","/api/v1/"+resource+"/"+object_id,token=token)[0]==404)

                # Two changes with the same old password must not both claim success.
                race_user=dict(user,email="race-password@example.com")
                race_token=expect(201,"POST","/register",race_user)["token"]
                barrier=Barrier(2)
                def password_parallel(i):
                    barrier.wait()
                    return call("POST","/change-password",{"old_password":user["password"],"new_password":"parallel-"+str(i)},race_token)[0]
                with ThreadPoolExecutor(max_workers=2) as pool:
                    statuses=list(pool.map(password_parallel,range(2)))
                login_results=[call("POST","/auth",{"email":race_user["email"],"password":"parallel-"+str(i)})[0] for i in range(2)]
                check("concurrent password changes validate current hash atomically",
                      sorted(statuses) == [200, 401] and login_results == statuses,
                      {"change_statuses": statuses, "subsequent_login_statuses": login_results})

                # Run the repository's real PostgreSQL tests against this disposable DB.
                test_env=dict(os.environ,AUTH_TEST_DB_DSN=env["AUTH_DB_DSN"])
                result=subprocess.run(["go","test","-race","-count=1","./..."],cwd=ROOT/"AuthService",env=test_env,text=True,capture_output=True)
                if result.returncode:
                    print(result.stdout+result.stderr)
                check("AuthService tests with real PostgreSQL and race detector",result.returncode==0,result.returncode)
                test_env["SHOP_TEST_DB_DSN"] = f"postgres://postgres@127.0.0.1:{db_port}/shopapi?sslmode=disable"
                result=subprocess.run(["go","test","-race","-count=1","./..."],cwd=ROOT/"ShopAPI",env=test_env,text=True,capture_output=True)
                if result.returncode:
                    print(result.stdout+result.stderr)
                check("ShopAPI PostgreSQL tests: concurrency and address rollback",result.returncode==0,result.returncode)


                address = sql("shopapi", "INSERT INTO addresses (country,city,street) VALUES ('RU','Moscow','Test') RETURNING id").splitlines()[0]
                category = sql("shopapi", "INSERT INTO categories (name) VALUES ('Appliances') RETURNING id").splitlines()[0]
                supplier = expect(201, "POST", "/api/v1/suppliers", {"name": "Supplier", "address_id": address,
                                  "phone_number": "+79990000000"}, token)["id"]
                product = expect(201, "POST", "/api/v1/products", {"name": "Kettle", "category_id": category,
                                 "supplier_id": supplier, "price": 100, "available_stock": 3}, token)
                expect(200, "GET", "/api/v1/products/" + product["id"], token=token)
                expect(204, "DELETE", "/api/v1/products/" + product["id"], token=token)
                expect(404, "GET", "/api/v1/products/" + product["id"], token=token)
                print("PASS authenticated store CRUD through gRPC middleware", flush=True)

                expect(401, "POST", "/change-password", {"old_password": "wrong", "new_password": "changed"}, token)
                expect(200, "POST", "/change-password", {"old_password": user["password"], "new_password": "changed"}, token)
                expect(401, "POST", "/auth", credentials)
                credentials["password"] = "changed"
                expect(200, "POST", "/auth", credentials)
                stop(auth)
                expect(401, "GET", "/api/v1/products", token=token)
                expect(503, "POST", "/auth", credentials)
                auth = subprocess.Popen([ROOT / "bin/auth"], env=env, stdout=auth_output, stderr=subprocess.STDOUT)
                ready()
                expect(200, "POST", "/auth", credentials)
                expect(200, "GET", "/api/v1/products", token=token)
                expect(200, "POST", "/reset", {"email": user["email"]})
                expect(401, "POST", "/auth", credentials)
                reset = re.search(r"Ваш новый пароль: ([^\r\n]+)", auth_log.read_text())
                assert reset, "email stub did not print the generated password"
                expect(200, "POST", "/auth", dict(credentials, password=reset.group(1)))
                expect(200, "POST", "/reset", {"email": "unknown@example.com"})
                print("PASS password change/reset, email stub, restart persistence and auth-service outage", flush=True)
        except Exception:
            # Auth logs can contain the reset password required by the assignment.
            # Keep the error output limited to non-secret startup diagnostics.
            if shop_log.exists():
                print(shop_log.read_text()[-4000:])
            raise
        finally:
            stop(shop)
            stop(auth)
            if pg_started:
                subprocess.run([pg_tool("pg_ctl"), "-D", str(data), "-m", "fast", "-w", "stop"],
                               check=True, stdout=subprocess.DEVNULL, env=pg_env)
            if report := os.environ.get("E2E_REPORT"):
                Path(report).write_text(json.dumps({"checks": checks, "requests": requests}, ensure_ascii=False, indent=2))
    passed=sum(result["passed"] for result in checks)
    failed=len(checks)-passed
    print(f"E2E RESULT: {passed} passed, {failed} failed; {len(requests)} HTTP requests. Temporary services and data removed.",flush=True)
    if failed:
        raise SystemExit(1)


if __name__ == "__main__":
    run()
