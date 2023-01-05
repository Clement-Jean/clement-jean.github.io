---
layout: post
author: Clement
title: Packed vs Unpacked Repeated Fields
categories: [Protocol Buffers]
image: /images/box.jpg
---

As this is a common and not well documented mistake that developers are doing, I decided to do a post explaining the problem that you might face when using repeated fields in your Protobuf messages.

Be sure to open any refresher section if you feel like you are not sure about a topic. We are going to use them during this post.

<p>
<details><summary><b>Refresher #1: Repeated Fields</b></summary>
<p>A repeated field is a field that can contain 0 or more values. In other words, this is a list. We can create such a field by simply adding a `repeated` modifier in front of the field. This looks like this:</p>
<p>
{% highlight proto %}
repeated int32 ids = 1;
{% endhighlight %}
</p>
</details>
</p>

<p>
<details><summary><b>Refresher #2: Field Options</b></summary>
<p>A field option is some additional information that will be affecting the compilation and thus the code generation. These options can be defined as key value pairs between square brackets between the field tag and the semicolon. In this post we are going to use the <code>packed</code> option, which takes a boolean as value and can only be used on repeated field. This looks like this:</p>
<p>
{% highlight proto %}
repeated int32 ids = 1 [packed = false];
{% endhighlight %}
</p>
</details>
</p>

<p>
<details><summary><b>Refresher #3: Protobuf Text Format</b></summary>
<p>Protobuf does not exclusively encode to binary. It is possible to encode to JSON or to a format that is close to JSON. This text format is generally used for improving readability/writeability (nobody want to read/write binary) and enhance your debugging or analysis of your messages. I will not go into too much details about this here, but to write a repeated field, you can simply repeated the field name as many times as you want to add value to the field, followed by a colon and the value. This looks like this:</p>
<p>
{% highlight yaml %}
ids: 1
ids: 2
ids: 3
{% endhighlight %}
</p>
</details>
</p>

</p>

## Packed

Let's start with packed repeated fields. In order to see how they are encoded we are going to use `protoc --encode` and pass it the content of some file defining the values in Protobuf Text Format. In this text file, let's define 3 values:

```yaml repeated.txt codeCopyEnabled
ids: 1
ids: 2
ids: 3
```

Then, for our proto file, we are going to store these values in a message called `PackedRepeated` that has a field of type `repeated int32`.

```proto repeated.proto codeCopyEnabled
syntax = "proto3";

message PackedRepeated {
  repeated int32 ids = 1;
}
```

And finally, we need to use the `--encode` flag from protoc, which let us take some binary content on the standard input and write some protobuf encoded message on the standard ouput. To take advantage of this we are going to display the content of a file on the standard output, pipe that to the standard input of protoc and finally, pipe the standard ouput of protoc to a command that display an hexadecimal dump.

```shell Linux/MacOS
$ cat repeated.txt | protoc --encode=PackedRepeated proto/repeated.proto | hexdump -C
00000000  0a 03 01 02 03                                    |.....|
00000005
```

```shell Windows (Powershell)
$ (Get-Content ./repeated.txt | protoc --encode=PackedRepeated proto/repeated.proto) -join "`n" | Format-Hex
   Label: /Users/clement/Git/experiment/out.bin

          Offset Bytes                                           Ascii
                 00 01 02 03 04 05 06 07 08 09 0A 0B 0C 0D 0E 0F
          ------ ----------------------------------------------- -----
0000000000000000 0A 03 01 02 03                                  �����
```

So here we can see that the end result of encoding `repeated.txt` content as `PackedRepeated` is `0A 03 01 02 03`. What does that mean? Let's decrypt that.

To do that, we can simply take each hexadecimal number and transform it into binary. While this is pretty simple numbers, let's use the command line to make sure we don't slip up and have wrong binary.

```shell Linux/MacOS
$ echo "ibase=16; obase=2; 0A" | bc
1010

$ echo "ibase=16; obase=2; 03" | bc
11

$ echo "ibase=16; obase=2; 01" | bc
1

$ echo "ibase=16; obase=2; 02" | bc
10
```

```shell Windows (Powershell)
$ [Convert]::ToString(0x0A, 2)
1010

$ [Convert]::ToString(0x03, 2)
11

$ [Convert]::ToString(0x01, 2)
1

$ [Convert]::ToString(0x02, 2)
10
```

> Note: When you are using integer that are not fixed, you are dealing with varints. This means that the bigger the value, the bigger the amount of bytes it will be encoded to. In our example, we purposely chose small numbers so that they are encoded into 1 byte. The following encoding explanation is not correct for all numbers you might use.

- `0A` gives us `1010`. This is a byte that represent both the wire type (type of value) and the field tag. To get the wire type, we simply take the first 3 bits starting from the right. In our case this is `010` or 2. if you check the [Encoding](https://developers.google.com/protocol-buffers/docs/encoding#structure) page of Protobuf Documentation, this means that we have a Length-Delimited type. In other words, we have some kind of data that has a dynamic size. This is exactly what we have, this is a list. Then, we are left with a tag equal to 1.
- `03` gives us `11`. This is the actual length of the list. Here we have 3 values.
- `01`, `02` and `03` (we omitted it, because we know the result), gives us respectively `1`, `10` and `11`. These are the actual values that we added into the list.

In the end, we have 5 bytes, 1 byte for type + tag, 1 byte for the list length, and 3 bytes for the values. Pretty compact.

## Unpacked

Let's now see how the same values are encoded in an unpacked repeated field. To do that, we are going to use the `packed` field option. We are going to set that to false so that protoc skip the packing.

```proto repeated.proto codeCopyEnabled
message UnpackedRepeated {
  repeated int32 ids = 1 [packed = false];
}
```

With that done, we can now run similar commands as what we did in the `Packed` section. The only difference is that, now, we need to specify that we want to encode the data as `UnpackedRepeated`.

```shell Linux/MacOS
$ cat repeated.txt | protoc --encode=UnpackedRepeated proto/repeated.proto | hexdump -C
00000000  08 01 08 02 08 03                                 |......|
00000006
```
```shell Windows (Powershell)
$ (Get-Content ./repeated.txt | protoc --encode=UnpackedRepeated proto/repeated.proto) -join "`n" | Format-Hex
   Label: String (System.String) <01DCACCB>

          Offset Bytes                                           Ascii
                 00 01 02 03 04 05 06 07 08 09 0A 0B 0C 0D 0E 0F
          ------ ----------------------------------------------- -----
0000000000000000 08 01 08 02 08 03                               ������
```

And ... We have 6 bytes.

There are two things we can notice here. The first is that now we don't have any `0A` byte. And the second one is that we are interleaving `08` with our values. Let's find out how this was encoded.

As we already know the values for `01`, `02` and `03`, we can just convert `08`.

```shell Linux/MacOS
$ echo "ibase=16; obase=2; 08" | bc
1000
```
```shell Windows (Powershell)
$ [Convert]::ToString(0x08, 2)
1000
```

- `08` gives us `1000`. Once again this is the combination of wire type and field tag. So we have 0 for the wire type, which correspond to varint. And then the field tag is 1.

So in this case, we are basically encoding each value of the list as a separate field. Protobuf will then see that the `ids` field is repeated and that we are adding multiple values with the same field tag and it will just add these values to the list.

In the end, Protobuf is encoding `UnpackedRepeated` into 6 bytes instead of 5. This sounds negligible here because we have a simple example but if you run the example on 100 ids:

> You can generate the repeated.txt by running this in your shell:
> ```shell Linux/MacOS codeCopyEnabled
> for i in {1..100}
> do
>   echo "ids: ${i}" >> repeated.txt
> done
> ```
> ```shell Windows (Powershell) codeCopyEnabled
> foreach ($i in 1..100) {
>   Add-Content -Path "repeated1.txt" -Value "ids: $i"
> }
> ```

```shell Linux/MacOS
$ cat repeated.txt | protoc --encode=PackedRepeated proto/repeated.proto | hexdump -C
00000000  0a 64 01 02 03 04 05 06  07 08 09 0a 0b 0c 0d 0e  |.d..............|
00000010  0f 10 11 12 13 14 15 16  17 18 19 1a 1b 1c 1d 1e  |................|
00000020  1f 20 21 22 23 24 25 26  27 28 29 2a 2b 2c 2d 2e  |. !"#$%&'()*+,-.|
00000030  2f 30 31 32 33 34 35 36  37 38 39 3a 3b 3c 3d 3e  |/0123456789:;<=>|
00000040  3f 40 41 42 43 44 45 46  47 48 49 4a 4b 4c 4d 4e  |?@ABCDEFGHIJKLMN|
00000050  4f 50 51 52 53 54 55 56  57 58 59 5a 5b 5c 5d 5e  |OPQRSTUVWXYZ[\]^|
00000060  5f 60 61 62 63 64                                 |_`abcd|
00000066

$ cat repeated.txt | protoc --encode=UnpackedRepeated proto/repeated.proto | hexdump -C
00000000  08 01 08 02 08 03 08 04  08 05 08 06 08 07 08 08  |................|
00000010  08 09 08 0a 08 0b 08 0c  08 0d 08 0e 08 0f 08 10  |................|
00000020  08 11 08 12 08 13 08 14  08 15 08 16 08 17 08 18  |................|
00000030  08 19 08 1a 08 1b 08 1c  08 1d 08 1e 08 1f 08 20  |............... |
00000040  08 21 08 22 08 23 08 24  08 25 08 26 08 27 08 28  |.!.".#.$.%.&.'.(|
00000050  08 29 08 2a 08 2b 08 2c  08 2d 08 2e 08 2f 08 30  |.).*.+.,.-.../.0|
00000060  08 31 08 32 08 33 08 34  08 35 08 36 08 37 08 38  |.1.2.3.4.5.6.7.8|
00000070  08 39 08 3a 08 3b 08 3c  08 3d 08 3e 08 3f 08 40  |.9.:.;.<.=.>.?.@|
00000080  08 41 08 42 08 43 08 44  08 45 08 46 08 47 08 48  |.A.B.C.D.E.F.G.H|
00000090  08 49 08 4a 08 4b 08 4c  08 4d 08 4e 08 4f 08 50  |.I.J.K.L.M.N.O.P|
000000a0  08 51 08 52 08 53 08 54  08 55 08 56 08 57 08 58  |.Q.R.S.T.U.V.W.X|
000000b0  08 59 08 5a 08 5b 08 5c  08 5d 08 5e 08 5f 08 60  |.Y.Z.[.\.].^._.`|
000000c0  08 61 08 62 08 63 08 64                           |.a.b.c.d|
000000c8
```
```shell Windows (Powershell)
$ (Get-Content ./repeated.txt | protoc --encode=PackedRepeated proto/repeated.proto) -join "`n" | Format-Hex
   Label: String (System.String) <470F6C47>

          Offset Bytes                                           Ascii
                 00 01 02 03 04 05 06 07 08 09 0A 0B 0C 0D 0E 0F
          ------ ----------------------------------------------- -----
0000000000000000 0A 64 01 02 03 04 05 06 07 08 09 0A 0B 0C 0A 0E �d��������������
0000000000000010 0F 10 11 12 13 14 15 16 17 18 19 1A 1B 1C 1D 1E ����������������
0000000000000020 1F 20 21 22 23 24 25 26 27 28 29 2A 2B 2C 2D 2E � !"#$%&'()*+,-.
0000000000000030 2F 30 31 32 33 34 35 36 37 38 39 3A 3B 3C 3D 3E /0123456789:;<=>
0000000000000040 3F 40 41 42 43 44 45 46 47 48 49 4A 4B 4C 4D 4E ?@ABCDEFGHIJKLMN
0000000000000050 4F 50 51 52 53 54 55 56 57 58 59 5A 5B 5C 5D 5E OPQRSTUVWXYZ[\]^
0000000000000060 5F 60 61 62 63 64                               _`abcd

$ (Get-Content ./repeated.txt | protoc --encode=UnpackedRepeated proto/repeated.proto) -join "`n" | Format-Hex
   Label: String (System.String) <6F5008AF>

          Offset Bytes                                           Ascii
                 00 01 02 03 04 05 06 07 08 09 0A 0B 0C 0D 0E 0F
          ------ ----------------------------------------------- -----
0000000000000000 08 01 08 02 08 03 08 04 08 05 08 06 08 07 08 08 ����������������
0000000000000010 08 09 08 0A 08 0B 08 0C 08 0A 08 0E 08 0F 08 10 ����������������
0000000000000020 08 11 08 12 08 13 08 14 08 15 08 16 08 17 08 18 ����������������
0000000000000030 08 19 08 1A 08 1B 08 1C 08 1D 08 1E 08 1F 08 20 ���������������
0000000000000040 08 21 08 22 08 23 08 24 08 25 08 26 08 27 08 28 �!�"�#�$�%�&�'�(
0000000000000050 08 29 08 2A 08 2B 08 2C 08 2D 08 2E 08 2F 08 30 �)�*�+�,�-�.�/�0
0000000000000060 08 31 08 32 08 33 08 34 08 35 08 36 08 37 08 38 �1�2�3�4�5�6�7�8
0000000000000070 08 39 08 3A 08 3B 08 3C 08 3D 08 3E 08 3F 08 40 �9�:�;�<�=�>�?�@
0000000000000080 08 41 08 42 08 43 08 44 08 45 08 46 08 47 08 48 �A�B�C�D�E�F�G�H
0000000000000090 08 49 08 4A 08 4B 08 4C 08 4D 08 4E 08 4F 08 50 �I�J�K�L�M�N�O�P
00000000000000A0 08 51 08 52 08 53 08 54 08 55 08 56 08 57 08 58 �Q�R�S�T�U�V�W�X
00000000000000B0 08 59 08 5A 08 5B 08 5C 08 5D 08 5E 08 5F 08 60 �Y�Z�[�\�]�^�_�`
00000000000000C0 08 61 08 62 08 63 08 64                         �a�b�c�d
```

you will get 102 bytes with the packed version and 200 with the unpacked one. Ouch!

## I'll never use `packed = false`, so what's the problem?

As of now, we were using an example that would probably never appear in real life. So now, it's time to get back in touch with reality. Let's say that instead of storing as `int32` you want to store your ids as strings. To test that, we can create a Simple message called `Repeated` with a repeated string field.

```proto repeated.proto codeCopyEnabled
message Repeated {
  repeated string ids = 1;
}
```

and change our text file to specify string values.

```yaml repeated.txt codeCopyEnabled
ids: "1"
ids: "2"
ids: "3"
```

After that, we are familiar how to encode that, we can just change the `--encode` flag value to `Repeated`.

```shell Linux/MacOS
$ cat repeated.txt | protoc --encode=Repeated proto/repeated.proto | hexdump -C
00000000  0a 01 31 0a 01 32 0a 01  33                       |..1..2..3|
00000009
```
```shell Windows (Powershell)
$ (Get-Content ./repeated.txt | protoc --encode=Repeated proto/repeated.proto) -join "`n" | Format-Hex

   Label: String (System.String) <7AB0A992>

          Offset Bytes                                           Ascii
                 00 01 02 03 04 05 06 07 08 09 0A 0B 0C 0D 0E 0F
          ------ ----------------------------------------------- -----
0000000000000000 0A 01 31 0A 01 32 0A 01 33                      ��1��2��3
```

Does it look familiar to you? Yes, we are interleaving `0A` (length-delimited type with tag 1) with the values (two bytes, `01` is the length and `31`, `32`, `33` are the ASCII values for `1`, `2`, `3`).

This is basically showing us that, even though repeated fields are packed by default, some types cannot be packed. This is the case for the following types:

- `bytes`
- `string`
- User defined Types (messages)

## Conclusion

The overall idea of this post was to explain that some types are not 'packable' when used in repeated fields. Simple types like varints and other numbers can be packed but more complex types cannot. This can cause performance problems and this can even result in poor performance compared to JSON. So the thing to keep in mind when using repeated field is that we should mostly use it with numbers. For other types, use `repeated` with caution.

**If you find this kind of article interesting or you would like me to cover some topic on Protobuf or gRPC, be sure to let me know in the comments.**
