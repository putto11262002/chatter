# Chatter
A real-time group messaging application that can be deployed anywhere with just a single binary file and a database file.

The primary goal of this project is to explore Go's features, including concurrency and design patterns, while gaining experience in building real-time systems.

A standout feature of this project is the real-time WebSocket infrastructure developed within the `ws` package. This infrastructure simplifies the creation of real-time, event-driven applications by abstracting away the complexities of WebSocket management. It allows developers to focus on implementing business logic within handlers that are bound to specific events, rather than dealing with the underlying WebSocket details.

## Features
- Real-Time Messaging
- Typing Indicators
- Read and Sent Notifications
- User Online Status

