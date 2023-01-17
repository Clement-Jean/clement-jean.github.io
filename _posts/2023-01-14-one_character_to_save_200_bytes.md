---
layout: post
author: Clement
title: One Character to Save 200 Bytes
categories: [Protocol Buffers]
---

Recently, I've been working with [Markus Tacker](https://techhub.social/@coderbyheart@chaos.social) on improving his [comparison of JSON vs. Protobuf for a Wifi Site Survey](https://github.com/coderbyheart/json-protobuf-comparison-wifi-site-survey). This has been a lot of fun and I thought I could do a simple post about what went well and what my mistakes were.

## Looking at the proto file

Here is the proto file at the moment I was looking at it:

```proto
syntax = "proto3";

message WiFiSiteSurvey {
  uint32 timestamp = 1;
  repeated AP accesspoints = 2;
}

message AP {
  int64 mac = 1;
  string ssid = 2;
  int32 rssi = 3;
  int32 channel = 4;
}
```

The first thing that made me think about improving this schema is the use of repeated on a complex object. If you don't know why, I go into more details about why it is less efficient to use complex objects in a repeated field, in the article called [Packed vs. Unpacked Repeated Fields](https://clement-jean.github.io/packed_vs_unpacked_repeated_fields/).

Other than that, I didn't have any other idea at that point. I needed to analyze the data.

## Analyzing the Data

The first thing to do when you are dealing with data is to understand it and get a sense of the possible values you can have. One thing that came out directly after running the comparison script and analyzing the [sitesurvey.json](https://github.com/coderbyheart/json-protobuf-comparison-wifi-site-survey/blob/saga/sitesurvey.json) is that all `rssi` properties are negative.

Now, I'm not an expert in wifi protocol but after searching online what `rssi` meant, I found that it's an acronym for Received Signal Strength Indicator and that it will always be a negative value ranging from -30 to -90 (see [here](https://corecabling.com/understanding-received-signal-strength-rssi-in-your-wifi-network/)).

This is interesting because by knowing this range we know that we are dealing with a range that fits in the 32 bytes integers and we know that the numbers are all negative so we will prefer to use a sint instead of an int (TODO: article about sint vs int). And, if you look at the proto file shown above the `rssi` field has the type `int32`, which means that we are encoding all the values into 10 bytes (because negative values are encoded as big positive numbers).

OK, so, before applying the change, Markus got the following result:

```shell
$ node compare.js
Found APs 30
JSON payload length: 1949 bytes
Protobuf payload length: 966 bytes
```

This is already very nice because we save 50% of bytes in our payload. But, after changing `int32 rssi = 3;` to `sint32 rssi = 3;`, we got the following result:

```shell
$ node compare.js
Found APs 30
JSON payload length: 1949 bytes
Protobuf payload length: 713 bytes
```

Two hundred+ bytes gone, with one character added. Pretty cool!

## Back to the Original Idea

Even though we saved 200 bytes, that wasn't my original idea on how to improve this proto file. As I mentioned I wanted to see if making the repeated field act on simple data could help.

Now, this is important to note that everything that comes after this wasn't added to the repository since we didn't entirely understand the requirements for the data. So I will show the assumption that we were making at that time and we will see how it was dismissed later. Here are the assumption:

- None of the fields are optional if an info is missing treat the data as erroneous.

This is important because with that assumption we could make multiple repeated fields instead of having the `AP` message and we would save encoding a complex object. This would be lowering the payload size and then later on because all the lists have the same length we could do a zip between these lists to get the objects back (first object get first element of all the lists). So the proto file changed like so:

```proto
syntax = "proto3";

message WiFiSiteSurvey {
  uint32 timestamp = 1;
  repeated int64 macs = 2;
  repeated string ssids = 3;
  repeated int32 rssis = 4;
  repeated int32 channels = 5;
}
```

> Note: `string` is a complex object so we are still using unpacked repeated field on `ssids`.

We first filtered all the erroneous data in the dataset and rerun the comparison. Here is the result:

```shell
$ node compare.js
Found APs 24
JSON payload length: 1949 bytes
Protobuf payload length: 510 bytes
```

Another 200 bytes gone.

## Why the repeated 'trick' didn't work

This mostly didn't work because some of the fields in `AP` are actually optional. This means that either we would have to add empty wrappers into the lists to get the lists have the same length (not worth, the payload size would be bigger than 713 bytes) or we go back to our `AP` message after the `sint32` improvement.

The second thing that is not making this approach work is that, if you noticed, we are lowering the payload but we are doing more computation in our code. We need to do a zip afterwards. This might be fine if this is internal to your company and well documented. However, if this is a client facing proto file, this might just make their life harder.

**Lesson: Know your data requirements!**

## Other Improvements

Here is a list of further improvements, added or not yet added, that are not impacting payload size:

- Change `uint32 timestamp = 1;` to `uint64 timestamp = 1;` for accepting a bigger range of numbers.
- Change `int32 channel = 4;` to `uint32 channel = 4;` for invalidating negative numbers on the client side.
- Change `int64 mac = 1;` to `uint64 mac = 1;` for invalidating negative numbers on the client side (I'm not sure yet, but this seems possible).

## Conclusion

We saw that by knowing your data and knowing the encoding algorithm behind Protobuf, we can get really big payload size improvements. However, we still need to care about the usage of our proto files and be more accurate on the different data requirements; otherwise we will implement obscure 'fixes' and in the end they will not be needed.

**If you like this kind of content let me know in the comment section and feel free to ask for help on similar projects, recommend the next post subject or simply send me your feedback.**