export class Filter {
  brokerName: string;
  queueName: string;
  messageId: string;
  correlationId: string;
  fromDate: Date;
  toDate: Date;
  messageBody: string;
  filteredHeaders: Map<string, string>
  criteriaSet: boolean;
  destinationQueueName: string;
  selectedMessageIDs: string[];
}