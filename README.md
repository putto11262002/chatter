# Chatter

Chatter is a real-time group messaging application designed to be deployed with remarkable simplicity, requiring only a single binary file and a database file.
The project serves as an exploration of Go's powerful features, particularly focusing on concurrency and design patterns while gaining hands-on experience in building real-time systems.

## Architecture

Chatter employs a modern, event-driven architecture that bridges real-time and traditional web communication technologies.
The application is constructed using a React frontend and a Go backend, strategically utilizing REST API and WebSocket protocols to deliver a seamless messaging experience.

The REST API handles non-real-time operations such as authentication, user profile management, and message history retrieval.
In contrast, WebSockets power the real-time communication layer, enabling instant message transmission, user status updates, and interactive features like typing indicators.

At the core of the system is an event-driven WebSocket implementation where each message is treated as a discrete event.
Custom event handlers are designed to process specific message types, allowing for modular and extensible real-time communication.
This approach enables complex interaction scenarios to be managed efficiently, with each event triggering targeted handlers that respond to different communication states and user interactions.

## Features

The application offers a comprehensive set of real-time communication capabilities including real-time messaging, typing indicators, read and sent notifications, and user online status tracking.

## Configuration

To begin using Chatter, start by configuring your environment. Copy the `config.example.yml` file to `config.yml` and update the values as needed.
The configuration options are detailed in [app/config.go](app/config.go), with the default configuration providing a functional out-of-the-box experience, though it's not recommended for production use.

Next, copy the `example.env` file to `.env` to configure the client settings.

## Development

For local development, run:

```bash
make dev
```

This command launches the Go server with hot reload and starts a VITE development server for the client.

## Deployment

To build the entire application into a single binary file:

```bash
make build
```

This generates a binary file containing both server and client components at `bin/chatter`. Run it simply with:

```bash
./bin/chatter
```

### Docker Deployment

Build the Docker image:

```bash
make docker-build
```

Run the Docker container:

```bash
docker run -p 8080:8080 chatter
```

## Licensing

The project is licensed under the MIT License. Detailed information can be found in the [LICENSE](LICENSE) file.
