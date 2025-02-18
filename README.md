# PROG2005-assignment-2

[TOC]

<br>
<br>
<br>

## Project description

### Brief overview

This is the project "Assignment 2" in the course PROG2005 at NTNU Gjøvik.

The end product of the project is a REST web application, where users can retrieve information about the percentage of renewable energy for (most of) the countries in the world.

The application depends on data about countries from the "REST Countries API"; for this assignment hosted locally at NTNU.

Furthermore, a "Renewable Energy Dataset" from https://ourworldindata.org/energy, in the form of a .csv document is used to find renewable energy percentages.

The application has endpoints for retrieving information about renewable energy percentages; both current and historic. Registering webhooks for getting notifications about renewable energy in countries of interest is also implemented. All responses from the service are in .json format.

The service is deployed as a Docker container in NTNUs OpenStack.

<br>
<br>

## Endpoints

The service has four main endpoints:

| Endpoint                      | Purpose/functionality                   |
| ----------------------------- |-----------------------------------------|
| /energy/v1/renewables/current | Current percentage of renewable energy  |
| /energy/v1/renewables/history | Historic percentage of renewable energy |
| /energy/v1/notifications/     | Register/view/delete webhooks           |
| /energy/v1/status/            | View status of the service              |

<br>
<br>

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
    ...
]
```

<br>
<br>

### Historic

The "history" endpoint shows the renewable energy percentage over more years.

For a single country, renewables percentage can be viewed year by year (within a specified range).

For multiple countries (no country input) renewable percentage is viewed as average for each country withing the specified range.

It is possible to use the name of the country (in English) instead of the country code. Both are case-insensitive.

General form of request:

```
Method: GET
Path: /energy/v1/renewables/history/{country?}{?begin=year&end=year?}{?sortByValue=bool?}
```

Where...

* {country?} refers to a country by name or 3-letter name code (optional)
* {?begin=year&end=year?} refers to a range of years from "begin" until "end" (optional). It is also possible to input either "begin" or "end" to get results from or to the specified year.
* {?sortByValue=bool?} specifies sorting of values by renewables percentage, ascending (true = sorting)

<br>
<br>

#### Example: year range, no country code:

```
/energy/v1/renewables/history/?begin=1990&end=1995
```

Corresponding response:

```json
[
    {
        "name": "Belarus",
        "isocode": "BLR",
        "percentage": 0.016133428706477087
    },
    {
        "name": "Japan",
        "isocode": "JPN",
        "percentage": 5.169685522715251
    },
    {
        "name": "Morocco",
        "isocode": "MAR",
        "percentage": 2.874749263127645
    },
    
    ...
    
]
```

<br>
<br>

#### Example request 2; country code, no year range:

```
/energy/v1/renewables/history/deu
```

Corresponding response:

```json
[
    {
        "name": "Germany",
        "isocode": "DEU",
        "year": 1965,
        "percentage": 1.614503026008606
    },
    {
        "name": "Germany",
        "isocode": "DEU",
        "year": 1966,
        "percentage": 1.7416129112243652
    },
    
    ...
    
]
```

<br>
<br>

#### Example: country code and year range:

```
/energy/v1/renewables/history/germany?begin=1990&end=1995
```

Corresponding response:

```json
[
    {
        "name": "Germany",
        "isocode": "DEU",
        "year": 1990,
        "percentage": 1.336940050125122
    },
    {
        "name": "Germany",
        "isocode": "DEU",
        "year": 1991,
        "percentage": 1.2709577083587646
    },
    {
        "name": "Germany",
        "isocode": "DEU",
        "year": 1992,
        "percentage": 1.5204551219940186
    },
    
    ...
    
]
```

<br>
<br>

#### Example: no country code; sort by value inside a year interval:

```
/energy/v1/renewables/history/?begin=1990&end=1995&sortByValue=true
```

Corresponding response (sample from end of list):

```json
[
    ...
    {
        "name": "Brazil",
        "isocode": "BRA",
        "percentage": 45.01850382486979
    },
    {
        "name": "Iceland",
        "isocode": "ISL",
        "percentage": 61.09769821166992
    },
    {
        "name": "Norway",
        "isocode": "NOR",
        "percentage": 71.12905375162761
    }
]
```

<br>
<br>

### Notifications

The "notifications" endpoint lets users register and delete webhooks for getting notifications about a particular country of interest. It is also possible to view one or all of the registered webhooks.

General form of request:
```
/energy/v1/notifications/{?id}
```

Where {?id} is the unique id of the webhook (optional for GET, required for DELETE, unavailable for POST)

<br>
<br>

#### POST: Registration

```
Method: POST
/energy/v1/notifications/
```

Body:

```json
{
   "url": "https://localhost:8080/client/",
   "country": "SWE",
   "calls": 10
}
```

Corresponding response (unique webhook id):

```
{
    "webhook_id": "<webhook_id_text>"
}
```

<br>
<br>

#### GET: View Specific Registration
Utilizing the GET method and specifying an ID will allow you to see registered
details of the webhook if it is found in the registry.

Example request:
```
Method: GET
/energy/v1/notifications/rSTz0uFnAGaUtaEHw3RH
```
Response body
```json
{
    "webhook_id": "rSTz0uFnAGaUtaEHw3RH",
    "url": "https://localhost:8080/client/some_path",
    "country": "SWE",
    "calls": 10
}
```

<br>
<br>

#### GET: View all registrations
If an ID is not specified, all registered webhooks will be returned in the response body.


Example request:
```
Method: GET
/energy/v1/notifications/
```
Response body:
```json
[     
    {
        "webhook_id": "1asdjb324b2oudas",
        "url": "https://localhost:8080/client/some_path",
        "country": "SWE",
        "calls": 2
    },
    {
        "webhook_id": "oj09ioi3983ejf2ion",
        "url": "https://mywebhookservice.com/somepath",
        "country": "NOR",
        "calls": 5
    },
    ...
]
```

<br>
<br>

#### DELETE: Deletion of registration
Registered webhooks can be deleted using the DELETE method along with a specified
webhook ID.

Example request:
```
Method: DELETE
/energy/v1/notifications/rSTz0uFnAGaUtaEHw3RH
```

Response Body:
```
no response, only http status 200 if deleted.
```


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
    "countries_api": "503 Service Unavailable",
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

You will need a machine running Ubuntu 22.04 LTS. Before doing any of the setup steps, be sure to update the system:

```
sudo apt update
sudo apt upgrade
```




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

This also installs the Docker Compose plugin.

We have tested and confirmed that using the "apt repository" install method described works in Ubuntu 22.04 LTS. Other methods and Ubuntu versions may work as well.

To eliminate the need of writing "sudo" (or be root user) for all docker commands, we can give the user "ubuntu" permission to use docker commands:

```
sudo groupadd docker
sudo usermod -aG docker ubuntu
```

NOTE: A new login to the machine will be necessary for the change to take place.


<br>

#### Docker Compose plugin

The Docker Compose plugin must be installed. Normally, it is installed along with the Docker engine. If Compose for some reason is not installed, use the following commands:

`sudo apt-get update
sudo apt-get install docker-compose-plugin`

Verify the installation using

`docker compose version`

<br>

#### Golang

Support for Golang must be  installed in order to compile the source code.

Download the archive: 

`wget https://go.dev/dl/go1.20.3.linux-amd64.tar.gz`

Next, remove (potentially) existing version and extract:

`sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.20.3.linux-amd64.tar.gz`  

Add /usr/local/go/bin to PATH environment variables:

`sudo nano $HOME/.profile`

Add the following as last line in the .profile file:

`export PATH=$PATH:/usr/local/go/bin`

For the change to take place, log out and then in to your machine.
Lastly, check the installation using:

`go version`



A guide is available for additional support: https://docs.docker.com/language/golang/

<br>

#### Required local files

The application relies on a Google Firestore database. To access the database, a service-account certificate must be present on the host machine. 

The certificate must have the filename "sha.json", and be located in /home/ubuntu/.secret. This directory will be mounted as a volume for the container. The certificate is project specific.

```
mkdir ~/.secret
```

Then, populate the .secret directory with your Firestore certificate, preferably using scp. Example assumes that your current directory is where your sha.json is located, and the file is copied into the newly created ".secret" directory:

```
sudo scp -i ~/.ssh/<yourSSHPrivKey> sha.json ubuntu@<docker vm IP address>:~/.secret
```

<br>

#### Network

The host machine must be connected to the Internet, and have an associated floating IP address. Furthermore, these ports must be open to traffic:

| Port | Direction | Purpose |
| ---- | --------- | ------- |
| 22   | ingress   | SSH     |
| 8080 | ingress   | http    |
|      |           |         |

<br>

### Build and deploy
The source code for the project must be downloaded to the machine used for deployment.

Clone the repository (you must have access to this repo in order to continue):

Using ssh (will require that host machine has a corresponding ssh key registered in GitLab):
```
git clone git@git.gvk.idi.ntnu.no:course/prog2005/prog2005-2023-workspace/even/assignment2.git 
```
Using https (if prompted for username/password, use your login credentials for GitLab):
```
git clone https://git.gvk.idi.ntnu.no/course/prog2005/prog2005-2023-workspace/even/assignment2.git
```

To build and deploy:
A shell script is added for convenience at first time deployment. Once the repo is cloned, navigate to the "assignment2" directory.
From there, run: 
```
./deploy.sh
```

The application will be built, and a default "config.yaml" file will be added in a "config" directory in 
your home/ubuntu directory. This config file can be altered (e.g. using nano) to make the application 
change behaviour. By default, the app is in development mode (running stub service), so you will need to change this
in the config. Navigate to "~/config/config.yaml" and change "development mode" to "false". 
To introduce the changes to the running application, navigate to "assignment2" directory and run:
```
docker compose restart
```
Note: If "deploy.sh" is run after updating the config file in "~/config", then the config file will be
overwritten by a default config file.


After first deployment, you can use the following commands to start, stop and remove containers:

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

<br>
<br>

## Implementation

### Persistence:
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
- The interval used to check against the DB is set with webhook-event-rate in the config file.
Note that the minimum time interval is 10 seconds. 
- As the synchronization with the DB and trigger check is 
done in bulk to avoid problems with concurrency the process can take in excess of 3 seconds depending on
the available resources.

<br>
<br>

## Testing

### Unit tests

Test coverage reports can be generated in html format, running the following commands from project root.

This command generates a coverage report:

```
go test -covermode=count -coverpkg=./... -coverprofile cover.out -v ./...
```

To view the report as html in the browser, use:

```
go tool cover -html "cover.out"
```

To generate a html file stored in the project directory, use:

```
go tool cover -html cover.out -o cover.html
```

The test coverage for our project is >80%. 

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



