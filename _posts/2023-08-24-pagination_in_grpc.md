---
layout: post
author: Clement
title: "Pagination in gRPC"
categories: [Go, gRPC]
---

Looking into optimizing one of my APIs, I recently stumbled upon the following resource: [Common design patterns](https://cloud.google.com/apis/design/design_patterns). This contains a lot of insights on how to design proto files in order to make gRPC APIs more idiomatic or more efficient. Within this document, I found the pagination part interesting and decided to write an article on how to implement it.

## A disclaimer

In this article I assume that you already know how to create a simple server in gRPC. **I will not show all the boilerplate, you can check it in [here](https://github.com/Clement-Jean/clement-jean.github.io/tree/working/src/2023-08-24-pagination_in_grpc)**.

Finally, this article contains data from [Packt](https://subscription.packtpub.com/search). They do not sponsor this article in any way. I'm not getting any money promoting these books (except mine), I simply needed some interesting data for this article.

## An explanation

Before starting the implementation, let's first understand what we are implementing.

### What

Pagination is a mechanism which allows the consumer of the API to get a subset of the available resources. This is done in order to limit the payload size returned by the API and thus make the API response faster.

This is generally implemented with the combination of `page_size` and `page_token` fields. The former tells how big is the subset we want, and the latter act as an index from which we are going to get the next subset.

Let's see an example of such a pagination. We have the following data:

```json
{
  "books": [
    {
      "name": "gRPC Go for Professionals",
      "description": "In recent years, the popularity of microservice architecture has surged, bringing forth a new set of requirements.",
      "authors": [
        "Clément Jean"
      ],
      "published": "2023-07-01T00:00:00Z",
      "pages": 260,
      "isbn": "9781837638840"
    },
    {
      "name": "Full-Stack Web Development with Go",
      "description": "Go is a modern programming language with capabilities to enable high-performance app development.",
      "authors": [
        "Nanik Tolaram",
        "Nick Glynn"
      ],
      "published": "2023-02-01T00:00:00Z",
      "pages": 302,
      "isbn": "9781803234199"
    },
    {
      "name": "Domain-Driven Design with Golang",
      "description": "Domain-driven design (DDD) is one of the most sought-after skills in the industry.",
      "authors": [
        "Matthew Boyle"
      ],
      "published": "2022-12-01T00:00:00Z",
      "pages": 204,
      "isbn": "9781804613450"
    },
    {
      "name": "Building Modern CLI Applications in Go",
      "description": "Although graphical user interfaces (GUIs) are intuitive and user-friendly, nothing beats a command-line interface",
      "authors": [
        "Marian Montagnino"
      ],
      "published": "2023-03-01T00:00:00Z",
      "pages": 406,
      "isbn": "9781804611654"
    },
    {
      "name": "Functional Programming in Go",
      "description": "While Go is a multi-paradigm language that gives you the option to choose whichever paradigm works best",
      "authors": [
        "Dylan Meeus"
      ],
      "published": "2023-03-01T00:00:00Z",
      "pages": 248,
      "isbn": "9781801811163"
    },
    {
      "name": "Event-Driven Architecture in Golang",
      "description": "Event-driven architecture in Golang is an approach used to develop applications that shares state changes asynchronously, internally, and externally using messages.",
      "authors": [
        "Michael Stack"
      ],
      "published": "2022-11-01T00:00:00Z",
      "pages": 384,
      "isbn": "9781803238012"
    },
    {
      "name": "Test-Driven Development in Go",
      "description": "Experienced developers understand the importance of designing a comprehensive testing strategy to ensure efficient shipping and maintaining services in production.",
      "authors": [
        "Adelina Simion"
      ],
      "published": "2023-04-01T00:00:00Z",
      "pages": 342,
      "isbn": "9781803247878"
    },
    {
      "name": "Mastering Go",
      "description": "Mastering Go is the essential guide to putting Go to work on real production systems.",
      "authors": [
        "Mihalis Tsoukalos"
      ],
      "published": "2021-08-01T00:00:00Z",
      "pages": 682,
      "isbn": "9781801079310"
    },
    {
      "name": "Network Automation with Go",
      "description": "Go’s built-in first-class concurrency mechanisms make it an ideal choice for long-lived low-bandwidth I/O operations, which are typical requirements of network automation and network operations applications.",
      "authors": [
        "Nicolas Leiva"
      ],
      "published": "2023-01-01T00:00:00Z",
      "pages": 442,
      "isbn": "9781800560925"
    },
    {
      "name": "Microservices with Go",
      "description": "This book covers the key benefits and common issues of microservices, helping you understand the problems microservice architecture helps to solve, the issues it usually introduces, and the ways to tackle them.",
      "authors": [
        "Alexander Shuiskov"
      ],
      "published": "2022-11-01T00:00:00Z",
      "pages": 328,
      "isbn": "9781804617007"
    },
    {
      "name": "Effective Concurrency in Go",
      "description": "The Go language has been gaining momentum due to its treatment of concurrency as a core language feature, making concurrent programming more accessible than ever.",
      "authors": [
        "Burak Serdar"
      ],
      "published": "2023-04-01T00:00:00Z",
      "pages": 212,
      "isbn": "9781804619070"
    }
  ]
}
```

As expected, if we start by requesting subset of size 2, we will get the first two books (`gRPC Go for Professionals` and `Full-Stack Web Development with Go`). On top of that result, a page token will be returned to us. If we now use this token and request a subset of size 2 we will get the 2 following books (`Domain-Driven Design with Golang` and `Building Modern CLI Applications in Go`).

This is pretty much it. It is simple to grasp and also simple to implement.

## The setup

In this article:

- I will be using Postgres to store our books' data. You can find the initialization script [here](https://github.com/Clement-Jean/clement-jean.github.io/tree/working/src/2023-08-24-pagination_in_grpc/db)
- I will run Postgres and the gRPC with Docker Compose. You can find the YAML file [here](https://github.com/Clement-Jean/clement-jean.github.io/tree/working/src/2023-08-24-pagination_in_grpc/docker-compose.yml)


## Protobuf

If you check the [List Pagination](https://cloud.google.com/apis/design/design_patterns#list_pagination) section, you will see that we have the following protobuf schema:

```proto
rpc ListBooks(ListBooksRequest) returns (ListBooksResponse);

message ListBooksRequest {
  string parent = 1;
  int32 page_size = 2;
  string page_token = 3;
}

message ListBooksResponse {
  repeated Book books = 1;
  string next_page_token = 2;
}
```

This code is mostly correct but we are going to remove the `parent` field. If you are interested in knowing what this is used for, check the [List Sub-Collections](https://cloud.google.com/apis/design/design_patterns#list_sub-collections) section.

So we now have:

```proto
message ListBooksRequest {
  int32 page_size = 1;
  string page_token = 2;
}

message ListBooksResponse {
  repeated Book books = 1;
  string next_page_token = 2;
}

service BookStoreService {
  rpc ListBooks(ListBooksRequest) returns (ListBooksResponse);
}
```

The last thing that we need to do is defining the `Book` message:

```proto
import "google/protobuf/timestamp.proto";

message Book {
  string name = 1;
  string description = 2;
  repeated string authors = 3;
  google.protobuf.Timestamp published = 4;
  uint32 pages = 5;
  string isbn = 6;
}
```

There is nothing fancy here. We simply laid out all the information needed to represent our books.

## ListBooks

Let's get started with an empty implementation for `ListBooks`:

```go
func (s *server) ListBooks(ctx context.Context, req *pb.ListBooksRequest) (*pb.ListBooksResponse, error) {
}
```

The first step in every endpoint implementation is to validate arguments. In our case we will validate the `page_size`. We mostly need to check that the `page_size` is not too big because it defeats the purpose of pagination, and if no `page_size` is provided we are going to set it to a default value.

```go
const (
  defaultPageSize = 10
  maxPageSize     = 30
)

func validatePageSize(req *pb.ListBooksRequest) error {
  if req.PageSize > maxPageSize {
    msg := fmt.Sprintf(
      "expected page size between 0 and %d, got %d",
      maxPageSize, req.PageSize,
    )
    return errors.New(msg)
  } else if req.PageSize == 0 { // no page_size provided
    req.PageSize = defaultPageSize
  }

  return nil
}
```

This means that we can now do the following in `ListBooks`:

```go
if err := validatePageSize(req); err != nil {
  return nil, status.New(codes.InvalidArgument, err.Error()).Err()
}
```

Next, we need to validate `page_token`. In this implementation, I decided to use [ULIDs](https://github.com/ulid/spec). This is because they are short ids and they are lexicographically sortable. The sortability part is interesting because we will sort the books by their IDs which are ULIDs.

Fortunately for us oklog provides an [ULID implementation](https://github.com/oklog/ulid) for us to verify if a ULID is valid or not. In `ListBooks`, we can simply do:

```go
if _, err := ulid.Parse(req.PageToken); len(req.PageToken) != 0 && err != nil {
  msg := fmt.Sprintf("expected valid ULID, got error %v", err)
  return nil, status.New(codes.InvalidArgument, msg).Err()
}
```

Notice that the `page_token` is optional (`len(req.PageToken) != 0`). When we do not provide one we will start from the beginning of the dataset.

Then, we need to generate the SQL query in order to get the subsets. We need to create the following SQL:

```sql
SELECT *
FROM book
WHERE id > page_token
LIMIT page_size
ORDER id ASC
```

Obviously, because the page_token is optional, the where clause is optional too.

Using [GORM](https://gorm.io/), we can easily create the request by writing the following:

```go
query := s.db.Table("book").Limit(int(req.PageSize)).Order("id ASC")

if len(req.PageToken) != 0 {
  query = query.Where("id > ?", req.PageToken)
}
```

Now that we have the query, we can simply execute it and map the result into our Protobuf `Book` model:

```go
var queryRes = []Book{}
query.Scan(&queryRes) // execute query

if len(queryRes) == 0 {
  // short circuit if not results
  return &pb.ListBooksResponse{}, nil
}

books := utils.Map(queryRes, mapBookToBookPb)
```

Finally, we can get the ID (ULID) of the last item in subset (`queryRes`) and this will represent the `page_token` from where a subsequent request need to start getting new result.

```go
lastItemIdx := len(queryRes) - 1
nextPageToken := queryRes[lastItemIdx].ID

if len(queryRes) < int(req.PageSize) {
  // no more pages
  nextPageToken = ""
}

return &pb.ListBooksResponse{
  Books:         books,
  NextPageToken: nextPageToken,
}, nil
```

And we now have pagination! Let's go ahead and test it.

## Testing

The first thing we can test is the case where the consumer doesn't provide a `page_token` and `page_size`. This should return 10 results (see `defaultPageSize`) from the beginning of the data.

```bash
$ grpcurl -plaintext \
          -proto proto/store.proto \
          -d '{}' \
          0.0.0.0:50051 BookStoreService.ListBooks

{
  "books": [
    {
      "name": "Full-Stack Web Development with Go",
      ...
    },
    {
      "name": "Domain-Driven Design with Golang",
      ...
    },
    {
      "name": "Building Modern CLI Applications in Go",
      ...
    },
    {
      "name": "Functional Programming in Go",
      ...
    },
    {
      "name": "Event-Driven Architecture in Golang",
      ...
    },
    {
      "name": "Test-Driven Development in Go",
      ...
    },
    {
      "name": "Mastering Go",
      ...
    },
    {
      "name": "Network Automation with Go",
      ...
    },
    {
      "name": "Microservices with Go",
      ...
    },
    {
      "name": "Effective Concurrency in Go",
      ...
    }
  ],
  "nextPageToken": "01H8EH4VYYCS6M4BFVZ90RP7FS"
}
```

First, you can notice that we had 11 datum and that because we asked for 10 we didn't get "gRPC Go for Professionals". And secondly, we can see that we got the `nextPageToken` field.

Let's now use the `nextPageToken` as `page_token`:

```bash
$ grpcurl -plaintext \
          -proto proto/store.proto \
          -d '{"page_token": "01H8EH4VYYCS6M4BFVZ90RP7FS"}' \
          0.0.0.0:50051 BookStoreService.ListBooks

{
  "books": [
    {
      "name": "gRPC Go for Professionals",
      ...
    }
  ]
}
```

And here we get our 11th datum!

Finally, we can try mixing the `page_token` and `page_size` fields. Let's say that we are going to have a `page_size` of 2. We will do the first request without `page_token` to get the 2 first elements:

```bash
$ grpcurl -plaintext \
          -proto proto/store.proto \
          -d '{"page_size": 2}' \
          0.0.0.0:50051 BookStoreService.ListBooks

{
  "books": [
    {
      "name": "Full-Stack Web Development with Go",
      ...
    },
    {
      "name": "Domain-Driven Design with Golang",
      ...
    }
  ],
  "nextPageToken": "01H8EH2RM7HVFJG4HYA4XTV0R5"
}
```

and then we can use the `nextPageToken` to get the 3rd and 4th elements:

```bash
$ grpcurl -plaintext \
          -proto proto/store.proto \
          -d '{"page_size": 2, "page_token": "01H8EH2RM7HVFJG4HYA4XTV0R5"}' \
          0.0.0.0:50051 BookStoreService.ListBooks

{
  "books": [
    {
      "name": "Building Modern CLI Applications in Go",
      ...
    },
    {
      "name": "Functional Programming in Go",
      ...
    }
  ],
  "nextPageToken": "01H8EH3CKPT5BX263G0NGGKQCQ"
}
```

Here we go! Everything workks as expected!

## Conclusion

We saw that we can implement pagination quite easily in gRPC with the combination of `page_token` and `page_size` fields in the request. We also saw that the API endpoint will return a `next_page_token` that we can later use as an index for the next page we want to get.

**If you like this kind of content let me know in the comment section and feel free to ask for help on similar projects, recommend the next post subject or simply send me your feedback.**