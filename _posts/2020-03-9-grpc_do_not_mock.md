---
layout: post
author: Clement
title: gRPC 'mocking'
categories: [gRPC, Android, Kotlin, Java]
---

After being used to the traditional way of debug an android app by using mocking and interceptors, I came across an interesting problem with gRPC. I wanted to do the same. Basically, add an interceptor that mock a server response.

**An important note: The app is in Kotlin, however this code I show is in Java. What I want to show here is that the concepts in gRPC are generally easily convertible to other languages. And even if you are actually coding in another supported language, you should be able to do pretty much the same**

I discovered that, of course, gRPC has interceptors, but I also discovered in my case I didn't need them. Let me explain here. After playing a little bit with the [ForwardingClientCall.SimpleForwardingClientCall](https://grpc.github.io/grpc-java/javadoc/io/grpc/ForwardingClientCall.SimpleForwardingClientCall.html) class by inheriting it like [this](https://github.com/grpc/grpc-java/blob/master/examples/src/test/java/io/grpc/examples/header/HeaderServerInterceptorTest.java):

{% highlight java %}

public void serverHeaderDeliveredToClient() {
    class SpyingClientInterceptor implements ClientInterceptor {
      ClientCall.Listener<?> spyListener;

      @Override
      public <ReqT, RespT> ClientCall<ReqT, RespT> interceptCall(
          MethodDescriptor<ReqT, RespT> method, CallOptions callOptions, Channel next) {
        return new SimpleForwardingClientCall<ReqT, RespT>(next.newCall(method, callOptions)) {
          @Override
          public void start(Listener<RespT> responseListener, Metadata headers) {
            spyListener = responseListener =
                mock(ClientCall.Listener.class, delegatesTo(responseListener));
            super.start(responseListener, headers);
          }
        };
      }
    }

{% endhighlight %}

I didn't find any way to send back a message to my client. After going through the gRPC code and especially the comments (ctrl+left click on android studio),something caught my eye:

{% highlight java %}

/**
    ...
 * <p>DO NOT MOCK: Use InProcessServerBuilder and make a test server instead.
 *
 * @param <ReqT> type of message sent one or more times to the server.
 * @param <RespT> type of message received one or more times from the server.
 */
public abstract class ClientCall<ReqT, RespT> {

{% endhighlight %}

Here was my solution: InProcessServerBuilder. After a little bit more searching in github, I found [this](https://github.com/grpc/grpc-java/blob/e6c8534f10d938566a62e38792a74032955e6c82/examples/src/test/java/io/grpc/examples/helloworld/HelloWorldServerTest.java):

{% highlight java %}

public void greeterImpl_replyMessage() throws Exception {
    // Generate a unique in-process server name.
    String serverName = InProcessServerBuilder.generateName();

    // Create a server, add service, start, and register for automatic graceful shutdown.
    grpcCleanup.register(InProcessServerBuilder
        .forName(serverName).directExecutor().addService(new GreeterImpl()).build().start());

    GreeterGrpc.GreeterBlockingStub blockingStub = GreeterGrpc.newBlockingStub(
        // Create a client channel and register for automatic graceful shutdown.
        grpcCleanup.register(InProcessChannelBuilder.forName(serverName).directExecutor().build()));


    HelloReply reply =
        blockingStub.sayHello(HelloRequest.newBuilder().setName( "test name").build());

    assertEquals("Hello test name", reply.getMessage());
  }
  {% endhighlight %}

  The most important thing to note here is the use of the two 2 builders:

  - InProcessServerBuilder
  - InProcessChannelBuilder

  and the use of InProcessServerBuilder's function :

  - addService

  And if you are familiar with the greeter example of gRPC, everything will make sense here. If you are not, you can join my course on [Udemy](addService) (my class is C# one, but trust me it is similar to Java/Kotlin).
  
  You can now adapt this solution for the testing part of you application !