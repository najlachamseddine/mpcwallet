# MPC WALLET

## Installation

```
go build
```

## Run the server

```
./mpcwallet
```

## Requests

### Create a wallet

```
curl http://localhost:8080/wallet
```

### Get the wallets

```
$ curl http://localhost:8080/wallets

["0xc04b990926d4c8a97ed667818c66be61fb90ba85"]
null

```


### Sign a data with a given wallet

```
$ curl "http://localhost:8080/sign?data=48656c6c6f2065766572796f6e65&wallet=0xc04b990926d4c8a97ed667818c66be61fb90ba85"

{"R":"4b8781d2251de91719517074826b657c9b312253fa637078ad8884c48e37388a","S":"3b36aa83241042d9ad34b56457241cd54fc592530dcbd441778c0a6ce28d3e65","Signature":"4b8781d2251de91719517074826b657c9b312253fa637078ad8884c48e37388a3b36aa83241042d9ad34b56457241cd54fc592530dcbd441778c0a6ce28d3e65"}

```
