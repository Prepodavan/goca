# Выпуск сертификата

Билд всего проекта с запуском
```
make && make run
```

OpenApi спецификация доступна в `/api` директории и по адресу `<address>/doc`.

Консольные параметры `go run cmd/issuer/main.go -help`

## Примеры

1. Выпуск по конфигу
```
$ curl -H "Accept: application/x-tar" -F "days=10"  -F "config=@server.conf"  --output - localhost:8080/issuing/cert > /tmp/archive.tar && tar -xvf /tmp/archive.tar
  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100 13310    0 12288  100  1022  67147   5584 --:--:-- --:--:-- --:--:-- 72732
server.conf
prv.pem
pub.pem
csr.pem
cert.pem
ca-cert.pem
```

2. Выпуск по csr
```
$ curl -H "Accept: application/zip" -F "days=10"  -F "csr=@csr.pem"  --output - localhost:8080/issuing/cert > /tmp/archive.zip && unzip /tmp/acrhive.zip
  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100  5071    0  3452  100  1619   306k   143k --:--:-- --:--:-- --:--:--  450k
Archive:  /tmp/archive.zip
  inflating: pub.pem                 
  inflating: csr.pem                 
  inflating: cert.pem                
  inflating: ca-cert.pem      
```