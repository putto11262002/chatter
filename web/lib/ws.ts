import { decodePacket, Packet } from "./ws/proto";

type WSOptions = {
  onReadStateChange: (readyState: ReadyState) => void;
  onPacketReceived: (packet: Packet) => void;
};

export enum ReadyState {
  Connecting = 0,
  Open = 1,
  Closing = 2,
  Closed = 3,
}

export class WS {
  private conn!: WebSocket;
  readyState: ReadyState = ReadyState.Closed;
  private onReadyStateChange: (readyState: ReadyState) => void;
  private onPacketReceived: (packet: Packet) => void;

  constructor({ onReadStateChange, onPacketReceived }: WSOptions) {
    this.onReadyStateChange = onReadStateChange;
    this.onPacketReceived = onPacketReceived;
  }

  connect() {
    if (this.readyState !== ReadyState.Closed) {
      return;
    }
    this.conn = new WebSocket("ws://localhost:8080/api/ws");
    this.readyState = ReadyState.Connecting;

    this.conn.addEventListener("open", () => {
      console.log("Connection opened");
      this.setReadyState(ReadyState.Open);
    });

    this.conn.addEventListener("message", (message) => {
      const packet = decodePacket(message.data);

      this.onPacketReceived(packet);
    });

    this.conn.addEventListener("close", (ev) => {
      console.log("Connection closed", ev);
      this.setReadyState(ReadyState.Closed);
    });

    this.conn.addEventListener("error", (error) => {
      console.log("Error", error);
      this.setReadyState(ReadyState.Closing);
      this.conn.close();
    });
  }

  setReadyState(readyState: ReadyState) {
    this.readyState = readyState;
    this.onReadyStateChange(this.readyState);
  }

  sendPacket(packet: Packet) {
    this.conn.send(JSON.stringify(packet));
  }

  public close() {
    this.conn.close();
  }
}
