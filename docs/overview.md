Overview
========

What is Santiago? Santiago is a web hook dispatching application.

Why would you need such a thing? Because Santiago tries hard for your Web Hooks to be properly dispatched. It offers an easy-to-use API that allows any of your applications to integrate seamlessly with each other.

Santiago is also very cloud-friendly and comes bundled with docker containers pre-built for both Production use as well as dev usage.

## Getting started

Let's get Santiago up and running in your machine in a few steps. For this quick start it is assumed you have [docker available and running](https://www.docker.com/products/docker).

First let's start a server that will receive our webhook. You can use anything you want for this, but we'll run an echo server in node.js:

    var http = require('http');
     
    http.createServer(function(request,response){
      response.writeHead(200);
      request.on('data',function(message){
        console.log("RECEIVED " + message);
        response.write(message);
      });

      request.on('end',function(){
        response.end();
      });
    }).listen(3000);

Now that we got our echo server, let's fire it up:

    $ node echo.js

Echo server is now running at 3000. Then in another terminal, let's test it:

    $ curl -dHello=World http://localhost:3000/

You should see `RECEIVED Hello=World` in the terminal running your echo server.

Now for the actual fun. Let's start our own Santiago server:

    $ docker pull tfgco/santiago-dev
    $ docker run -i -t --rm -p 8080:8080 tfgco/santiago-dev

Then let's enqueue a web hook to be dispatched in our Santiago server. For this part you'll need to know your network adapter IP address. You can find it out with this command:

    $ ifconfig | egrep inet | egrep -v inet6 | egrep -v 127.0.0.1 | awk ' { print $2 } '

Now that you know your IP, just replace $IP with your actual IP address:

    $ curl -dHello=World http://localhost:8080/hooks?method=POST&url=http%3A//$IP%3A3000/

Once more you should see `RECEIVED Hello=World` in the terminal running Santiago.

When you decide to run your Santiago app in production, please read our [Hosting] docs.

## Features

* **Reliable** - Santiago is very simple and relies on Redis for its queueing system;
* **Delivery Retry** - Santiago will retry up to 10 times to deliver your web hook (configurable ammount);
* **Log-Friendly** - We log almost any operation we do in Santiago, so you can easily debug it.
* **Easy to deploy** - Santiago comes with containers already exported to docker hub for every single of our successful builds. Just pick your choice!

## Architecture

Whenever you add a new web hook to Santiago, it enqueues it with Redis. There are workers running that process this queue and try to send your web hooks.

If the web hook fail, it re-enqueues the message up to a max number of times.

That's pretty much all there's to know about Santiago's architecture. Running redis is out of the scope of this document.

## The Stack

For the devs out there, our code is in Go, but more specifically:

* Web Framework - [Iris](https://www.gitbook.com/book/kataras/iris/details) based on the insanely fast [FastHTTP](https://github.com/valyala/fasthttp);
* Queueing - [Redis](http://redis.io).

## Who's Using it

Well, right now, only us at TFG Co, are using it, but it would be great to get a community around the project. Hope to hear from you guys soon!

## How To Contribute?

Just the usual: Fork, Hack, Pull Request. Rinse and Repeat. Also don't forget to include tests and docs (we are very fond of both).
