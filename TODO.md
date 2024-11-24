# TODO

## Hub

- [x] Instead of exposing channels, export functions that send messages to the channel

- [x] When a user conntects to the hub async the user to the hub

  - [ ] send some config ? e.g. maximum message size,
  - [x] send friends statuses

- [ ] Impose limit on the number of connections to the server
      https://stackoverflow.com/questions/22625367/how-to-limit-the-connections-count-of-an-http-server-implemented-in-go

- [ ] Impose rate limit on the server

- [ ] Implement graceful shotdown for server.

  - the the websocket connection is not shutdown automatically when calling server.Shotdown
    because the connection has ben hijacked by the websocket handler.
  - Munual closing of the connection is required. We can do this via hub.Close() which will
    attemp to close all connection gracefully within a deadline. If the deadline is reached, the
    connection will be forcefully closed.

- [ ] Impose a timeout to each operation in the hub via context

- [ ] Make the hub only responsible for routing messages.
  - move all the message parsing to the client. It is the client's responsible to ensure the incoming packs are valid
  - move all the database operations from the hub. The hub should spawn a new go routine to handle the database operations.
    Once these database operations is done send back the response to the hub via a channel and the hub will broadcast the response to the clients.

## API

- [ ] sign in should also return the user so that it can be used to populate the client state

## Chat Store

- [ ] Add read interaction for sender when sending a message
- [ ] Implement pagination for messages, rooms

## Frontend

- [ ] Implement infinite scrolling for messages and rooms

## Optimizations

- [x] Store last read message in room_user so when the user is reading messages we don't have to scan through the whole messages table only have to scan from the last onwards

- [ ] Do not have to store read in message_interaction just store a pointer to the last message read in room_user.
      By doing so we are assuming that the user would have read all the previous messages before sending a message This may not be the most provide the most accurate read status but it is efficient.

- [ ] Buffer pools

  - https://blog.cloudflare.com/recycling-memory-buffers-in-go/
  - https://www.captaincodeman.com/golang-buffer-pool-gotcha
  - https://brunocalza.me/how-buffer-pool-works-an-implementation-in-go/

- Profile the application to find bottlenecks
  - https://go.dev/blog/pprof
  - https://github.com/samonzeweb/profilinggo
  - https://github.com/DataDog/go-profiler-notes/blob/main/guide/README.md

## Benchmark

### Reading List

- https://bravenewgeek.com/benchmarking-message-queue-latency/

- https://medium.com/@srinathperera/how-do-we-measure-the-performance-of-a-microservice-service-or-server-450c562854a7

- https://jvm-gaming.org/t/how-to-properly-benchmark-server-performance/54489
