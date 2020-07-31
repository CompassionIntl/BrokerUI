# Broker Mobile

Broker Mobile Ionic is a single page application (SPA) developed in Angular and Ionic handling the frontend of the Broker UI dashboard.
It sends broker, queue, and message requests to Broker Service.

You can add as many servers as you need to <code>/adapters</code>.
Currently, Broker Service only supports ActiveMQ, RabbitMQ, and SQS servers.
The service sets up three endpoints in each adapter for message operations: get, move, and delete.

***

## Getting Started
Below are instructions to get a local copy up and running.

### Prerequisites
Install Node.js in broker-mobile-ionic.
<pre><code>npm install</code></pre>

### Run
Start up Angular.
<pre><code>ng serve</code></pre>

You might see Angular rendering chunks but eventually you should see a message like:

```
** Angular Live Development Server is listening on localhost:4200, open your browser on http://localhost:4200/ **
ℹ ｢wdm｣: Compiled successfully.
```

At this point you can go to the localhost specified: in our example: http://localhost:4200/.
If the broker list is empty, follow the instructions below.

### Start Broker Service
To retrieve brokers, you need to start Broker Service.
Navigate to <code>broker-service</code> in the Broker UI directory and follow the README.md setup instructions.


### Start local ActiveMQ Server
If you are using ActiveMQ and want to setup a localhost server, follow these [directions](https://activemq.apache.org/version-5-getting-started.html).

***

## Usage
Broker UI is a dashboard used to manage messages if you have multiple broker services running.
You can get, move, and delete messages from supported message queues.

### Features

When viewing messages in Broker UI, the dashboard will parse a message header and dynamically generate filter fields.
This allows you to easily sort or filter by standard or custom message fields.

![Image of Header Filters](https://i.imgur.com/ZTW755j.png)


You can click on a message to view the message headers with ease.

![Image of Headers](https://i.imgur.com/jfR1ImF.png)

***

### Environments
So far we have only been interacting with localhost. To setup Broker UI in devint, stage, or production you need to use environments.

Below is an example to setup a production environment.

1. Create a new environment in <code>/src/environments/</code> named <code>environment.prod.ts</code>
2. Paste the following code with your production URL replaced.
``` ts
    export const environment = {
    production: true,
    APIEndpoint: "<production-url>"
    };
```

In <code>angular.json</code> we support devint, stage, and production.