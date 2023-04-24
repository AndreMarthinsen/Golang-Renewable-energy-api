# PROG2005-assignment-2

[TOC]

<br>
<br>
<br>

## Project description

### Brief overview

This is the project "Assignment 2" in the course PROG2005 at NTNU Gjøvik.

The end product of the project will be a REST web application, where users can retrieve information about the percentage of renewable energy for (most of) the countries in the world.

The application depends data about countries from the "REST Countries API"; for this assignment hosted locally at NTNU.

Furthermore, a "Renewable Energy Dataset" from https://ourworldindata.org/energy, in the form of a .csv document will be used to find renewable energy percentages.

There will be endpoints for retrieving information about renewable energy, both current and historic. Registering webhooks for getting notifications about renewable energy in countries of interest will also be implemented. All responses will be in .json format.

The service will be deployed as a Docker container in NTNUs OpenStack.

<br>
<br>

## Endpoints

The service has four main endpoints:

| Endpoint                      | Purpose/functionality                   |
| ----------------------------- | --------------------------------------- |
| /energy/v1/renewables/current | Current percentage of renwable energy   |
| /energy/v1/renewables/history | Historic percentage of renewable energy |
| /energy/v1/notifications/     | Register/view/delete webhooks           |
| /energy/v1/status/            | View status of the service              |

### Current

The "current" endpoint provides renewable energy percentage for one or more countries for the most recent year in the dataset.

General form of request:

```
Method: GET
Path: /energy/v1/renewables/current/{country?}{?neighbours=bool?}
```

Where...

* {country?} refers to a country by 3-letter name code (optional)
* {?neighbours=bool?} determines if energy data for neighbouring countries should be retrieved (optional; requires preceding country code)

Example request 1; country code:

```
/energy/v1/renewables/current/nor
```

Corresponding response:

```json
[
    {
        "name": "Norway",
        "isocode": "NOR",
        "year": 2021,
        "percentage": 71.55836486816406
    }
]
```

Example request 2; country and neighbours:

```
/energy/v1/renewables/current/nor?neighbours=true
```

Corresponding response:

```json
[
    {
        "name": "Norway",
        "isocode": "NOR",
        "year": 2021,
        "percentage": 71.55836486816406
    },
    {
        "name": "Finland",
        "isocode": "FIN",
        "year": 2021,
        "percentage": 34.611289978027344
    },
    {
        "name": "Sweden",
        "isocode": "SWE",
        "year": 2021,
        "percentage": 50.924007415771484
    },
    {
        "name": "Russia",
        "isocode": "RUS",
        "year": 2021,
        "percentage": 6.620289325714111
    }
]
```

Example request 3; no country code:

```
/energy/v1/renewables/current
```

Corresponding response:

```json
[
    {
        "name": "Japan",
        "isocode": "JPN",
        "year": 2021,
        "percentage": 11.428995132446289
    },
    {
        "name": "Spain",
        "isocode": "ESP",
        "year": 2021,
        "percentage": 22.341663360595703
    },
    {
        "name": "Iran",
        "isocode": "IRN",
        "year": 2021,
        "percentage": 1.2903937101364136
    },
    
    .... all countries in dataset
    
]
```

<br>
<br>
<br>

### Historic


<br>
<br>
<br>

### Notifications

<br>
<br>
<br>

### Status

The "status" endpoint displays the status of the service. 

Example request:

```
/energy/v1/status
```

Corresponding response:

```json
{
    "countries_api": "501 Not Implemented",
    "notification_db": "200 OK",
    "webhooks": "2",
    "version": "v1",
    "uptime": 5029
}
```

This shows:

- status of REST Countries API

- status of Firebase (for storing webhooks and caching)

- number of registered webhooks

- current version of service

- uptime (in seconds) since last service restart

<br>
<br>
<br>

## Deployment
The application is designed to be deployed as a container using Docker in a linux environment.


### Configuration of Service
The service itself utilizes a config file for setting in-memory to firebase DB synchronization intervals, among others. 
See Implementation section for more details on time-intervals.


Example config.yaml:
```yaml
# Zero values for time settings will be overridden with setting defaults.
time-intervals:
    # time in seconds between each time updates to in-memory cache will be pushed to firebase DB
    # default: 5
  cache-push-rate: 5
    # time in minutes deciding how old a country cache entry can be before it is discarded
    # default: 60
  cache-time-limit: 60
    # time in seconds between each time registered webhooks are checked for trigger events.
    # Note: values < 10 will be overridden to 10.
    #
    # default: 10
  webhook-event-rate: 10

# settings for turning on and off internal development/deployment settings
deployment-variables:
    # setting debug-mode true leads to extra logging of events. Leave off for deployment.
  debug-mode: false
    # setting development-mode true causes the service to utilize stubbing of third party APIs.
    # WARNING: Set false for deployment, otherwise service will not function as intended.
  development-mode: true

# firebase paths and other variables
firebase-variables:
    # Name of the caching collection in the related firebase DB
  caching-collection-name: "Caches"
    # Name of the main cache document. If the document cannot be found, a new document will be
    # created and the cache will be stored to it.
  primary-cache-document-name: "TestData"
    # Name of the webhook collection in the firestore DB.
  webhook-collection-name: "Webhooks"
```

<br>
<br>
<br>


### Service Dependencies

The following must be in place in order to deploy the application as a Docker container on a virtual machine.

#### RESTCountries API
The service relies on an instance of the RESTCountries API for retrieving information about the bordering countries
of a country specified in certain API endpoints. It is recommended that you run your own instance by  
getting the source code from https://gitlab.com/restcountries/restcountries, rather than relying on sending traffic
to their currently hosted instance, both for the sake of ensuring stability of your service, and reducing load
on their service.

#### Firebase
The service relies on Google firebase for persistent storage of webhooks and neighbour information retrieved
from the third party API. See https://firebase.google.com/ for details on how to set up your own database for use
with your deployed service.

<br>
<br>

### Setup Dependencies

#### Docker engine

Docker engine must be installed on a virtual or physical machine running Ubuntu.
Follow this instruction to get the latest Docker version:
https://docs.docker.com/engine/install/ubuntu/

We have tested and confirmed that using the "apt repository" install method described works in Ubuntu 22.04 LTS. Other methods and Ubuntu versions may work as well.

#### Docker Compose plugin

The Docker Compose plugin must be installed. Use following commands:

`sudo apt-get update
sudo apt-get install docker-compose-plugin`

Verify the installation using

`docker compose version`

#### Golang

Support for Golang must be  installed in order to compile the source code.

Download the archive: 

`wget https://go.dev/dl/go1.20.3.linux-amd64.tar.gz`

Next, remove (potetntially) existing version and extract:

`rm -rf /usr/local/go && tar -C /usr/local -xzf go1.20.3.linux-amd64.tar.gz`  

Add /usr/local/go/bin to PATH environment variables:

nano $HOME/.profile

Add the following as last line in the .profile file:

`export PATH=$PATH:/usr/local/go/bin`

Lastly, check the installation using:

`go version`



A guide is available for additional support: https://docs.docker.com/language/golang/



#### Required local files

The application relies on a Google Firestore database. To access the database, a service-account certificate must be present on the host machine. 

The certificate must have the filename "sha.json", and be located in /home/ubuntu/.secret. This directory will be mounted as a volume for the container. The certificate is project specific.

#### Network

The host machine must be connected to the Internet, and have an associated floating IP address. Furthermore, these ports must be open to traffic:

| Port | Direction | Purpose |
| ---- | --------- | ------- |
| 22   | ingress   | SSH     |
| 8080 | ingress   | http    |
|      |           |         |

### Build and deploy
The source code for the project must be downloaded to the machine used for deployment.

Clone the repository:

```
git clone git@git.gvk.idi.ntnu.no:course/prog2005/prog2005-2023-workspace/even/assignment2.git 
```

To build and deploy the application, navigate to the project directory (assignment2) and run:
```
docker compose up -d
```

The application should now be running. A message should confirm that the container is started.

The service is set up to restart automatically when the host machine is rebooted.

To manually stop the service, this command can be used:

`docker stop <name-of-container>`

Using "stop" will stop the service until service is started again, or until next reboot of host machine.

To manually start a stopped container:

`docker start <name-of-container>`

To shut down the service and remove the container, this command can be used:

`docker compose down`

The service will not start again at reboot after using the "down" command.

## Implementation

### Persistence
#### RESTCountries Caching
The service makes use of in-memory caching of data retrieved from the third party RESTCountries API,
which in turn is periodically synchronized with a firebase DB with an interval determined by 
the cache-push-rate setting in the config.yaml file in project root. While the service will attempt
values as low as 1 second it is generally recommended to use larger intervals. The DB will be synchronized
on shut-down.

Entries older than the duration set by cache-time-limit will be purged on read/write of the cache.

- Cache entries are stored in the collection named by the caching-collection-name variable.
The collection contains the primary working cache and a backup which is created on the first read
on service start-up.
- The main cache document is named by the primary-cache-document-name in the config file.
Upon the initial read from the external DB on service boot, a backup is created of the previous
cache file.

#### Webhooks
Webhooks are stored persistently in the set firebase until deleted by a client that holds its ID.

The service counts up invocations of specific countries in the service API in a dedicated worker
thread which periodically will check stored webhooks to see if enough invocations have occurred to trigger
an event.
- The interval used to check against the DB is set with webhoook-event-rate in the config file.
Note that the minimum time interval is 10 seconds. 
- As the synchronization with the DB and trigger check is 
done in bulk to avoid problems with concurrency the process can take in excess of 3 seconds depending on
the available resources.

<br>
<br>
<br>

## Directory Structure

````
root
│   .env
│   .gitignore
│   .gitlab-ci.yml
│   compose.yml
│   Dockerfile
│   go.mod
│   go.sum
│   README.md
│
├───.github
│   └───workflows
│           deployment.yaml
│
├───caching
│       cache_worker.go
│       caching_structs.go
│       caching_util.go
│       invocation_worker.go
│
├───cmd
│       server.go
│       sha.json
│
├───consts
│       consts.go
│
├───Documentation_Internal
│       conventions.md
│       team-work.md
│       Workflow.drawio
│
├───fsutils
│       fsutils.go
│
├───handlers
│   │   renewables.go
│   │   status.go
│   │
│   └───notifications
│           notification.go
│           notification_structs.go
│
├───internal
│   ├───assets
│   │       codes=CHN.json
│   │       codes=FIN.json
│   │       codes=INV.json
|   |       ...
│   │       renewable-share-energy.csv
│   │
│   ├───stubbing
│   │       stubbing.go
│   │
│   └───testing
│       │   caching_test.go
│       │   fsutils_test.go
│       │   renewables_test.go
│       │   sha.json
│       │   stubbing_test.go
│       │   util_test.go
│       │
│       └───internal
│           └───assets ... copy of assets for testing
│
└───util
        dataset.go
        util.go
````



