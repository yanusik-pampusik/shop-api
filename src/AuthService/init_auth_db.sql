-- Run with psql connected to the maintenance database (postgres).
-- The auth database is separate from the store's shopapi database.
CREATE DATABASE shopapi_auth WITH ENCODING = 'UTF8' TEMPLATE = template0;

-- AuthService creates/migrates the users table in shopapi_auth on startup.
