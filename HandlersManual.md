# PROG2005-assignment-2

[TOC]



## Endpoint Usage

The service has four main endpoints:

| Endpoint                      | Purpose/functionality                   |
| ----------------------------- | --------------------------------------- |
| /energy/v1/renewables/current | Current percentage of renwable energy   |
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

It is possible to use the name of the country (in English) instead of the country code. Both are case insensitive.

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


