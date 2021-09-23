openssl genrsa -out server.key 2048
openssl req -new -batch -x509 -sha256 -key server.key -out server.crt -days 3650