# Broker Mobile

Broker Mobile Ionic is a single page application (SPA) developed in Angular and Ionic 
handling the frontend of the Broker UI dashboard.
It sends broker, queue, and message requests to Broker Service.

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
Navigate to <code>broker-service</code> in the Broker UI directory and follow the 
[README.md](https://github.com/CompassionIntl/BrokerUI/tree/master/broker-service#run) setup instructions.


### Start local ActiveMQ Server
If you are using ActiveMQ and want to setup a localhost server, follow these 
[directions](https://activemq.apache.org/version-5-getting-started.html).

***

### Environments
So far we have only been interacting with localhost. 
To setup Broker UI in devint, stage, or production you need to use environments.

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