#!/bin/bash
mkdir -p /c/nginx/ssl
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout /c/nginx/ssl/shop.key -out /c/nginx/ssl/shop.crt \
  -subj "/C=RU/ST=Moscow/L=Moscow/O=ShopAPI/CN=shop.local"