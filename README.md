# Chirpy

A toy server built in go. Uses a local json as DB (I know, sue me!)

### How to run

Hit the following cmd:

```sh
sh runServer.sh
```

And play around with the endpoints.
<br/>

Or else use the following to run the server in Debug mode. This nukes the database.

```sh
sh runServer.sh --debug
```

### Endpoints

- [GET] `/api/healthz` : Check the health of the server

- [GET] `/api/metrics` : Check hit metrics of server

- [GET] `/app` : Homepage

- [GET] `/app/assets` : Fileserver for assets

- [GET] `/api/chirps` : Get all the chirps in the DB

- [GET] `/api/chirps/{id}` : Get a particular chirp in the DB

- [POST] `/api/chirps` : Create a new chirp in the DB

- [GET] `/api/users` : Get all the users in the DB

- [GET] `/api/users/{id}` : Get a particular user in the DB

- [POST] `/api/users` : Create a new user in the DB
