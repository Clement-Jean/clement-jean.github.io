---
layout: post
author: Clement
title: If Modified Since for Redis
categories: [Redis]
---

Caching is everywhere! It is an essential part of most applications out there and so obviously there are a lot of options you can chose from. Here is a non exhaustive list:

- [Redis](https://redis.com)
- [Memcached](http://memcached.org)
- CDNs
- and lot more

And caching is very specific to the kind of data you are transfering (JSON, video, ...) and to your architecture.

**That's a lot of choices to make !**

At [E4E](http://educationforethiopia.org), since we are a stratup we can't take the risk to over engineer this. I will cost us time, money and make our architecture way harder to maintain. So we developed a simple [Redis plugin](https://github.com/Clement-Jean/RedisIMS) to help us with caching.

## Background

As I said, caching is very specific to your solution, there is no One size fits all solution. So let's see what our solution is providing first.

At E4E we provide educational video content for students in Ethiopia through an native Android app called [Saquama](https://play.google.com/store/apps/details?id=com.e4e.saquama). Every video comes with some metadata like: Title, Description and all the relational part that comes with it. For this article we are focusing on these metadata because videos are already taken care of by a CDN.

## What about the plugin?

RedisIMS standing for Redis If Modified Since (very creative, isn't it ?), provides a [HTTP protocol's If Modified Since Header](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/If-Modified-Since) like experience. This basically means that we have a timestamp saying when our metadata about a certain video changed. We can then compare this timestamp with the one sent by the client and returns a value accordingly.

### How does that work ? Technically I mean.

This plugin is heavily influenced by the following Lua code in this [article](https://blog.r4um.net/2021/redis-mtime-getset/#:~:text=Redis%20server%20side%20if-modified-since%20caching%20pattern%20using%20lua,can%20save%20significant%20network%20bandwidth%20and%20compute%20cycles.). The process consist in the following actions:

- When caching some data, the plugin will do a HSET of the key defined in the plugin, the data and the timestamp.

- When getting cached data, the plugin will use HGET with the key defined and the timestamp.
    - If the data doesn't exist, return NULL
    - If the data exists and the timestamp is bigger or equal than the cached one, we return NULL
    - and If the timestamp is smaller than the cached one, we return the cached data


## An example

{% highlight shell %}
redisims.exists MY_NON_EXISTING_KEY -> 0 
redisims.get MY_NON_EXISTING_KEY TIMESTAMP -> NULL

redisims.set MY_EXISTING_KEY THE_VALUE THE_TIMESTAMP
redisims.exists MY_EXISTING_KEY -> 1

redisims.get MY_EXISTING_KEY OUTDATED_TIMESTAMP -> YOUR_OBJECT
redisims.get MY_EXISTING_KEY CURRENT_TIMESTAMP -> NULL
{% endhighlight %}

## Interested ?

If you feel like contributing to the project or just trying it, head up to the [Github repository](https://github.com/Clement-Jean/RedisIMS).

And finnaly if you have any constructive feedback, feel free to reach me by checking the contact page of either my [Github profile](https://github.com/Clement-Jean) or the [about page of the website](https://clement-jean.github.io/about/)

