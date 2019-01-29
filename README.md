# CONVERT-Service
Convert an incoming JSON (or a different one in the future) request from AUTH service and convert it into GRPC request and send it to ROUTER service.

## Technologies
Service uses next technologies:
* SocketIO client (for connect to a config service and receive config)

## Features
* Now just convert a request from JSON to GRPC formats.
* Every incoming request must start with `/api` prefix.
* After it there is a GRPC connection pool to ROUTER service and create a new GRPC request of type `google.protobuf.Struct`. Information about requested method packed into GRPC header with key `proxy_method_name`. All authorization headers start with `x-` also packs into headers with the same names. Eventually, the request sends to ROUTING service `BackendService.Request`.
* **TODO.** To have abilities to accept an incoming request in different formats (GRPC, XML, etc.).
* **TODO.** To have abilities to return a response in different formats (GRPC, XML, etc.).
* **TODO.** To have abilities to balance proxying further request at different ROUTING service with different algorithms.

## Environment variables
`APP_PROFILE` - Name for config file, default is `config`. Will seek file with name: `config.yml`.

`APP_CONFIG_PATH` - Absolute path where a config file is. If a variable hadn't been specified config will seek into the same directory, when binary file placed.

`LOG_LEVEL` - Log level, the default is `INFO`.

`APP_MODE` - If set to `dev` logger for SQL will be on and if `LOG_LEVEL` hadn't been specified it will set to `DEBUG`.

## Setup
* There are necessary parameters you must specify in `config.yml` or what ever name does it has.

 ```
    configServiceAddress:
      ip: "10.250.9.114"
      port: "5000"
    moduleName: "converter"
    instanceUuid: "bf482806-0c3d-4e0d-b9d4-12c037b12d70"
 ```

## Usage
* There is also swagger: `http://127.0.0.1:5000/swagger/` and it's possible to call any method.

## Remote config example
```json
{
    "restAddress": {
        "ip": "0.0.0.0", "port": "5000"
    }, 
    "routerAddress": {
        "ip": "10.250.101.180", "port": "7001"
    }
}
```