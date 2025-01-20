# WS Hub Architecture

## Componnets

There are four important components in this websocket server architecture:

- Connection: the connection is a wrapper around the websocket connection. It provide abstractions to interact with the connection,
  such as sending/receiving messages, and closing the connection.
- Hub: the hub is responsible for managing the connections.

## Interaction between components

Understanding the interaction between components is important to understand how the server works. We
will go through different scenarios to understand the interaction between components.

### Client open connection to the server

The webscoket client opens a connection to the server by sending a HTTP request with the `Upgrade` header set to `websocket`.
Within the handler for the HTTP request, the `ConnFactory` is used to create a new implementation of `Conn` interface.
The created `Conn` is passed to `hub.Connect` method. The `hub.Connect` should be non-blocking. Its responsibilities are:

1. Starting the `Conn.readLoop` and `Conn.writeLoop` goroutines.
   It is the hub responsibility to track and ensure that the goroutines are properly closed when the connection is closed.
2. Store the connection in some type of data structure which allow easy look up by the connection id.

### Client closes the server gracefully

> Both the client and the web server can initiate the closing handshake. Upon receiving a close frame, an endpoint (client or server) has to send a close frame as a response (echoing the status code received). After an endpoint has both sent and received a close frame, the closing handshake is complete, and the WebSocket connection is considered closed.

In the case that that client wishes to close the connection. It sends a close message to the server, the sever then
responds with a close message and closes the connection.

### Server closes the connection gracefully
