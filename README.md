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
Return a wallet object (address, public key, and keys datas of the Participants in the keygen). It might take some time (1 minute) to get the response back. The newly created wallets are stored in a map. This map in memory is cleaned if the server is restarted. To have persistent wallets data, these wallets must be stored in a database.

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
