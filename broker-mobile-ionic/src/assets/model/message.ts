export class Message {
  Queue: string;
  MessageID: string;
  Timestamp: number;
  Body: string;
  Headers: Map<string, string>;
  CorrelationID: string;
  isChecked: boolean;
}

export class MessageUpdate {
  messageIDs: string[];
}