# Broker Service

Broker Service connects Broker UI to message queue servers.

You can add as many servers as you need to <code>/adapters</code>.
Currently, Broker Service only supports ActiveMQ, RabbitMQ, and SQS servers.
The service sets up three endpoints in each adapter for message operations: get, move, and delete.

***

### Localhost Setup
Run the <code>prime-env.sh</code> script to set up your environment variables.  
The basic setup assumes you have ActiveMQ running locally.  You must run the script
with a preceding '. ./' (dot-space-dot-slash) in order for your environment variables 
to stay in the current terminal context.

<pre><code>. ./prime-env.sh</code></pre>

***

### Run
You need to build and run main.go.

<pre>
<code>go build</code>
<code>./broker-service</code>
</pre>

***

## How to Configure or Add Brokers

#### Environment Variable Configuration (default)
The application will scan your environment for the following variables:

<pre>
BROKER1_USER
BROKER1_PASS
BROKER1_NAME
BROKER1_TYPE
</pre>

With these values, it will attempt to create the first adapter.
It will also look for subsequent configurations in your environment variables with the
following pattern:

<pre>
BROKER#_USER
BROKER#_PASS
BROKER#_NAME
BROKER#_TYPE
</pre>

The '#' will be replaced with sequential numbers up to 100.  Once it discovers that a given 
'BROKER#' is not in the environment, it will discontinue looking for additional adapter 
configurations.

#### Other Configuration Managers
At this time, only the Environment Variable Configuration Manager is available.  However,
additional configuration manager can be implemented by complying to the `configuration/ConfigurationManager`
interface. 
***
### Endpoints and Examples

The following are endpoints setup in each adapter file.

#### List Broker Names
>GET - /brokers/

#### List Queues for a Broker
>GET - /brokers/[broker]/queues/

#### List Messages in a Queue
>GET - /brokers/[broker]/queues/[queue]/messages

#### Move a Message from Queue to Queue (Same Server)
>POST - /brokers/[broker]/queues/[queue]/toqueue/[queue]/messages/[messageid] 

#### Move Multiple Messages from Queue to Queue
>POST - /brokers/[broker]/queues/[queue]/toqueue/[queue]/messages 

Body:
<pre>
"messageIDs" :
[
    messageId,
    messageId,
    ...
]
</pre>

#### Purge Queue
>DELETE - /brokers/[broker]/queues/[queue]

#### Delete Message from Queue
>DELETE - /brokers/[broker]/queues/[queue]/message/[messageid]

#### Delete Multiple Messages from Queue
>DELETE - /brokers/[broker]/queues/[queue]/messages 

Body:
<pre>
"messageIDs" :
[
    messageId,
    messageId,
    ...
]
</pre>

***

### ActiveMQ Properties

The ActiveMQ adapter supports multiple broker and console URLs.
In <code>prime-env.sh</code>, add multiple URLs separated by a comma.
The adapter will try to connect to the URLs in order until one is successful.

Currently, ActiveMQ is the only adapter with test cases.
Run <code>activeMQAdapter_test.go</code> to test the adapter.