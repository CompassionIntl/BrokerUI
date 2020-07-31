import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Observable, of } from 'rxjs';
import { messages } from '../assets/mockData/messages';
import { Message, MessageUpdate } from 'src/assets/model/message.js';
import { queues } from '../assets/mockData/queues';
import { brokers } from '../assets/mockData/brokers';
import { Queue } from 'src/assets/model/queue.js';
import { Filter } from 'src/assets/model/filter.js';
import { Broker } from 'src/assets/model/broker';
import { Platform } from '@ionic/angular';
import { environment } from './../environments/environment';

@Injectable({
  providedIn: 'root'
})
export class BrokerService {
  serviceURL: string;
  headers: HttpHeaders;
  messages: Message[];
  brokers: Broker[];
  queues: Queue[];
  numOfReturned: number;
  filter: Filter;
  selectedMessage: Message;

  constructor(private http: HttpClient, private platform: Platform) {
    // this.serviceURL = "http://C02Y50J7JGH7:1323";
    // if (this.platform.is("android")) {
    //   this.serviceURL = "http://10.0.2.2:1323";
    // } else {
    //this.serviceURL = "http://localhost:1355";
    // }
    //this.serviceURL = "https://bridgetest.ci.org/broker-service"
    //this.serviceURL = "https://bridgestage.ci.org/broker-service"
    //this.serviceURL = "https://bridgeprod.ci.org/broker-service"

    this.serviceURL = environment.APIEndpoint;

    this.headers = new HttpHeaders()
      .set("Content-Type", "application/json")
      .set("Access-Control-Allow-Origin", "*")
      .set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
      .set("Access-Control-Allow-Headers", "Origin, Content-Type, X-Auth-Token");
    // this.messages = messages;
    // this.queues = queues;
    //this.brokers = [];
    //this.brokers = brokers;
    this.numOfReturned = 0;
    this.filter = new Filter();
    this.filter.criteriaSet = false;
  }

  //filter methods

  setFilter(filter: Filter): void {
    this.filter = filter;
  }

  getFilter(): Filter {
    return this.filter;
  }

  //broker methods
  getBrokers(): Observable<Broker[]> {
    console.log("calling pulsar service...")
    return this.http.get<Broker[]>(this.serviceURL + "/brokers", { headers: this.headers });
  }

  setBrokers(brokers: Broker[]): void {
    this.brokers = brokers;
  }

  getSubscribedBrokers(): Broker[] {
    return this.brokers;
  }

  getMockBrokers(): Observable<Broker[]> {
    return of(this.brokers);
  }

  //queue methods
  getQueues(): Observable<Queue[]> {
    //TODO Add specific Broker to the service call
    return this.http.get<Queue[]>(this.serviceURL + "/brokers/" + this.filter.brokerName + "/queues", { headers: this.headers });
  }

  setQueues(queues: Queue[]): void {
    this.queues = queues;
  }

  getSubscribedQueues(): Queue[] {
    return this.queues;
  }

  getMockQueues(): Observable<Queue[]> {
    return of(this.queues);
  }

  //message methods
  //TODO api service returns a null response for empty queues. Hopefully real response of no messages found can be used in the future
  getMessages(): Observable<Message[]> {
    let encodedQueueName = encodeURIComponent(this.filter.queueName);
    return this.http.get<Message[]>(this.serviceURL + "/brokers/" + this.filter.brokerName + "/queues/" + encodedQueueName + "/messages", { headers: this.headers });
  }

  getMockMessages(): Observable<Message[]> {
    let filteredMessages = this.messages;
    if (this.filter.queueName != undefined) {
      filteredMessages = this.messages.filter(x => x.Queue == this.filter.queueName);
    }
    return of(filteredMessages);
  }

  setSelectedMessage(message: Message) {
    this.selectedMessage = message;
  }

  getSelectedMessage(): Message {
    return this.selectedMessage;
  }

  //service methods
  moveMessage(messageID: string): Observable<Message[]> {
    let encodedQueueName = encodeURIComponent(this.filter.queueName);
    let encodedDestinationQueueName = encodeURIComponent(this.filter.destinationQueueName);
    return this.http.post<Message[]>(this.serviceURL + "/brokers/" + this.filter.brokerName + "/queues/" + encodedQueueName + "/toqueue/" + encodedDestinationQueueName + "/messages/" + messageID, { headers: this.headers });
  }

  moveMessages(): Observable<Message[]> {
    let messageUpdate = new MessageUpdate();
    messageUpdate.messageIDs = this.filter.selectedMessageIDs;
    let encodedQueueName = encodeURIComponent(this.filter.queueName);
    let encodedDestinationQueueName = encodeURIComponent(this.filter.destinationQueueName);
    return this.http.post<Message[]>(this.serviceURL + "/brokers/" + this.filter.brokerName + "/queues/" + encodedQueueName + "/toqueue/" + encodedDestinationQueueName + "/messages", messageUpdate, { headers: this.headers });
  }

  //TODO After our deletes do we want to just refresh the page or something else?
  deleteMessage(messageID: string): Observable<Message[]> {
    let encodedQueueName = encodeURIComponent(this.filter.queueName);
    return this.http.delete<Message[]>(this.serviceURL + "/brokers/" + this.filter.brokerName + "/queues/" + encodedQueueName + "/messages/" + messageID, { headers: this.headers });
  }

  deleteMessages(): Observable<Message[]> {
    let messagesToDelete = new MessageUpdate();
    messagesToDelete.messageIDs = this.filter.selectedMessageIDs;
    let encodedQueueName = encodeURIComponent(this.filter.queueName);
    return this.http.request<Message[]>("delete", this.serviceURL + "/brokers/" + this.filter.brokerName + "/queues/" + encodedQueueName + "/messages", { body: messagesToDelete, headers: this.headers });
  }

  deleteAllMessages(queueName: string): Observable<Message[]> {
    let encodedQueueName = encodeURIComponent(queueName);
    return this.http.delete<Message[]>(this.serviceURL + "/brokers/" + this.filter.brokerName + "/queues/" + encodedQueueName, { headers: this.headers });
  }

  // show toast with message
  showToast(message: string, alert: boolean) {
    // Get the snackbar DIV
    let x = <HTMLElement>document.getElementById("snackbar");
    x.innerHTML = '' + message;
    if (alert) {
      x.style.background = 'red';
      x.style.color = 'white';
    } else {
      x.style.background = '#333';
      x.style.color = 'white';
    }
    // Add the "show" class to DIV
    x.className = "show";
    // After 3 seconds, remove the show class from DIV
    setTimeout(function () { x.className = x.className.replace("show", ""); }, 3000);
  }
}
