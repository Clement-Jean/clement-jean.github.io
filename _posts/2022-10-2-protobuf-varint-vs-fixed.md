---
layout: post
author: Clement
title: Protocol Buffers varint vs fixed
categories: [Protocol Buffers]
excerpt_separator: <!--desc-->
---

This article is much more a note to myself than something else but this might be interesting for people out there.

I wanted to calculate the thresholds at which it is better it is to use a `fixed` rather than a varint. <!--desc--> Now, knowing that the varint are encoded in base 128, this basically means that we are dealing with power of 128. This gives us the following table:

<div class="table-responsive">
<table class="table table-striped table-borderless">
  <thead>
    <tr>
      <th scope="col" class="text-center">Threshold value</th>
      <th scope="col" class="text-center">Bytes size (without tag)</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <th scope="row" class="text-center">0</th>
      <td class="text-center">0</td>
    </tr>
    <tr>
      <th scope="row" class="text-center">1</th>
      <td class="text-center">1</td>
    </tr>
    <tr>
      <th scope="row" class="text-center">128</th>
      <td class="text-center">2</td>
    </tr>
		<tr>
      <th scope="row" class="text-center">16,384</th>
      <td class="text-center">3</td>
    </tr>
		<tr>
      <th scope="row" class="text-center">2,097,152</th>
      <td class="text-center">4</td>
    </tr>
		<tr>
      <th scope="row" class="text-center">268,435,456</th>
      <td class="text-center">5</td>
    </tr>
		<tr>
      <th scope="row" class="text-center">34,359,738,368</th>
      <td class="text-center">6</td>
    </tr>
		<tr>
      <th scope="row" class="text-center">4,398,046,511,104</th>
      <td class="text-center">7</td>
    </tr>
		<tr>
      <th scope="row" class="text-center">562,949,953,421,312</th>
      <td class="text-center">8</td>
    </tr>
		<tr>
      <th scope="row" class="text-center">72,057,594,037,927,936</th>
      <td class="text-center">9</td>
    </tr>
  </tbody>
</table>
</div>

In summary:

- From 268,435,456 to whatever limit you 32 bits type has, it is better to use a `fixed32`.
- From 72,057,594,037,927,936 to whatever limit you 64 bits type has, it is better to use a `fixed64`.