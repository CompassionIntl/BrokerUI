import { Component, OnInit } from '@angular/core';

import { Message } from 'src/assets/model/message';
import { Pulse } from 'src/assets/model/pulse';
import { BrokerService } from '../broker.service';
import { Filter } from 'src/assets/model/filter';
import { Queue } from 'src/assets/model/queue';
import { Router, Params, ActivatedRoute} from '@angular/router';
import { MessageDetailsComponent } from '../message-details/message-details.component';
import { ModalController } from '@ionic/angular';
import { timeout } from 'rxjs/operators';

// Max time the client waits for messages in milliseconds
const getMessageTimeout = 60000;

@Component({
  selector: 'app-message',
  templateUrl: './message.component.html',
  styleUrls: ['./message.component.scss'],
})
export class MessageComponent implements OnInit {
  title = 'brokerUI';
  messages: Message[];
  filteredMessages: Message[];
  continue: boolean;
  messagesToDisplay: Map<string, Message[]>;
  messagesConsumed: Map<string, Message[]>;
  messagesPublished: Message[];
  authorsMessages: Pulse[];
  modalMessage: Message;
  filter: Filter;
  queues: Queue[];
  allChecked: boolean;
  queueTos: Queue[];

  filterRows: string[][];
  headerInputs: string[];
  headerNames: string[];

  isLoading = false;

  constructor(private brokerService: BrokerService, private router: Router,
    private route: ActivatedRoute, private modalController: ModalController) {
    
    this.messagesToDisplay = new Map<string, Message[]>();
    this.messages = [];
    this.allChecked = false;
  }

  ngOnInit() {
    
    this.queues = this.brokerService.getSubscribedQueues();
    this.filter = this.brokerService.getFilter();
    var hasRunGetMsg = this.validateFilters();
    
    this.filter.selectedMessageIDs = [];
    
    if (this.filter.brokerName !== undefined) {
      if (this.filter.queueName !== undefined) {
        if(this.queues != null && !hasRunGetMsg) 
          this.getMessages();
        
      } else {
        this.brokerService.showToast('Please select a queue first', true);
        this.router.navigate(['brokers', this.filter.brokerName, 'queues']);
      }
    } else {
      this.brokerService.showToast('Please select a broker first', true);
      this.router.navigate(['./brokers']);
    }
  }

  // Verify filters are setup correctly, used for manual URL routing.
  // Returns true if getMessages() has been executed, false otherwise.
  validateFilters(): boolean {
    var flag = false;
    if(this.filter.brokerName == null || this.queues == null) {
      this.route.paramMap.subscribe(params => {
        this.filter.queueName = params.get('queue');
      });
      
      this.brokerService.getQueues().subscribe(result => {
        if(result.filter(x => x.Name == this.filter.queueName).length == 0)
          return this.requestError();

        this.queues = result;
        this.brokerService.setQueues(this.queues);

        this.getMessages();
        flag = true;
      },
      error => {
        this.requestError();
      });
    }

    return flag;
  }

  private requestError() {
    this.brokerService.showToast('Invalid URL, please ensure broker and queue name are correct', true);
    this.router.navigate(['./brokers']);
  }

  getMessages() {
    this.queueTos = this.queues.filter(x => x.Name != this.filter.queueName);
    this.brokerService.getMessages().pipe(timeout(getMessageTimeout)).subscribe(result => {
      this.isLoading = true;
      //this.brokerService.getMockMessages().subscribe(result => {
      if (result) {
        this.messages = result;
        this.filteredMessages = this.messages;

        this.setupFilters();
        this.sort();

      } else {
        this.messages = [];
        this.filteredMessages = this.messages;
        this.brokerService.showToast("No messages found in " + this.filter.queueName, false);
      }
    }, (error) => {
      this.isLoading = true;
      this.brokerService.showToast(error, true);
      console.log('Error', error);
    });
    this.isLoading = false;
  }

  setupFilters() {
    this.filterRows = new Array<string[]>();
    this.headerNames = new Array();
    
    // Used to check for duplicate headers
    let headerSet = new Set<string>();
    let currRow = new Array();

    let [ col, row, k ] = [ 0, 0, 0 ];

    for (var i = 0; i < this.messages.length; i++) {
      let message = this.messages[i];
      for (var key of Object.keys(message.Headers)) {
        if(headerSet.has(key)) {
          continue;
        }

        headerSet.add(key);
        this.headerNames[k++] = key;

        if(col >= 3) {
          col = 0;
          this.filterRows[row++] = currRow;
          currRow = new Array();
        }

        currRow[col++] = key
      }
    }
    
    this.filterRows[row] = currRow;
    this.headerInputs = new Array(this.headerNames.length);
  }

  changeQueue() {
    this.messages = [];
    this.filter.selectedMessageIDs = [];
    this.brokerService.setFilter(this.filter);
    this.getMessages();
  }

  changeDestinationQueue() {
    this.brokerService.setFilter(this.filter);
  }

  clickMessage(message: Message) {
    this.modalMessage = message;
    (<HTMLElement>document.getElementById("messageModal")).style.display = "block";
  }

  clearFilters() {
    //Save the broker name and current queue before clearing the filters, should we save queue as well?
    let brokerName = this.filter.brokerName;
    let queueName = this.filter.queueName;
    this.headerInputs = new Array(this.headerNames.length);
    this.filter = new Filter();
    this.filter.brokerName = brokerName;
    this.filter.queueName = queueName;
    this.filteredMessages = this.messages;
    this.sort();
  }

  filterMessages() {
    this.filteredMessages = this.messages;
    let fromDate = new Date(this.filter.fromDate);
    fromDate.setHours(fromDate.getHours() + 24);
    let toDate = new Date(this.filter.toDate);
    toDate.setHours(toDate.getHours() + 24);

    if (this.filter.messageId !== undefined) {
      this.filteredMessages = this.filteredMessages.filter(x => x.MessageID.includes(this.filter.messageId));
    }
    if (this.filter.messageBody !== undefined) {
      this.filteredMessages = this.filteredMessages.filter(x => x.Body.includes(this.filter.messageBody));
    }
    if (this.filter.fromDate !== undefined) {
      this.filteredMessages = this.filteredMessages.filter(x => new Date(x.Timestamp) >= fromDate);
    }
    if (this.filter.toDate !== undefined) {
      this.filteredMessages = this.filteredMessages.filter(x => new Date(x.Timestamp) <= toDate);
    }
    
    // build map and filter headers that contain information
    this.filter.filteredHeaders = new Map<string, string>();
    for (var i = 0; i < this.headerNames.length; i++) {
      if(this.headerInputs[i] !== undefined) {
        this.filter.filteredHeaders.set(this.headerNames[i], this.headerInputs[i]);
        this.filteredMessages = this.filteredMessages.filter(x => x.Headers[this.headerNames[i]].includes(this.headerInputs[i]));
      }
    }

    this.sort();
  }

  selectMessage(message: Message) {
    this.brokerService.setSelectedMessage(message);
    this.router.navigate(['brokers', this.filter.brokerName, 'queues', this.filter.queueName, 'messages', 'message-details'], { skipLocationChange: true });
  }

  updateCheckedMessages(message: Message) {
    if (message.isChecked) {
      this.filter.selectedMessageIDs.push(message.MessageID);
      console.log("adding " + message.MessageID);
    } else {
      this.filter.selectedMessageIDs = this.filter.selectedMessageIDs.filter(m => m !== message.MessageID);
      console.log("removed " + message.MessageID);
    }
    this.brokerService.setFilter(this.filter);
  }

  moveMessages() {
    if (this.filter.destinationQueueName === undefined || this.filter.destinationQueueName === this.filter.queueName || this.filter.selectedMessageIDs.length === 0) {
      this.brokerService.showToast("Please choose a queue to move message(s) to! \n Destination queue can not be the same as the current queue \n Select atleast one message.", true);
    } else {
      let confirmed = confirm("Are you sure you want to move " + this.filter.selectedMessageIDs.length + " messages from: " + this.filter.queueName + " to: " + this.filter.destinationQueueName + "?");
      if (confirmed) {
        this.brokerService.moveMessages().subscribe(result => {
          //TODO Check result and don't always show success toast
          this.messages = this.messages.filter((msg) => { return !this.filter.selectedMessageIDs.includes(msg.MessageID) });
          this.filterMessages();
          console.log(result);
        });
        this.brokerService.showToast("Messages moved!", false);
      } else {
        console.log("Cancelled Move");
      }
    }
  }

  deleteMessages() {
    let confirmed = confirm("Are you sure you want to delete " + this.filter.selectedMessageIDs.length + " message(s)?");
    if (confirmed) {
      this.brokerService.deleteMessages().subscribe(result => {
        //TODO Check result and don't always show success toast
        this.messages = this.messages.filter((msg) => { return !this.filter.selectedMessageIDs.includes(msg.MessageID) });
        this.filterMessages();
        console.log(result);
      });
      this.brokerService.showToast("Messages deleted!", false);
    } else {
      console.log("Cancelled Delete");
    }
  }

  async presentModal(message: Message) {
    const modal = await this.modalController.create({
      component: MessageDetailsComponent,
      backdropDismiss: true,
      animated: true,
      showBackdrop: true,
      componentProps: {
        'message': message,
        'modalCtrl': this.modalController
      }
    });

    modal.onDidDismiss().then((data) => {
      let shouldRemoveMessage = data['data']['deletedOrMoved']
      if (shouldRemoveMessage) {
        let index = this.messages.indexOf(message)
        if (index > -1) {
          this.messages.splice(index, 1);
        }
      }
    });
    return await modal.present();
  }

  checkAll() {
    this.allChecked = !this.allChecked;
    if (this.allChecked) {
      for (let message of this.messages) {
        message.isChecked = true;
        this.updateCheckedMessages(message);
      }
    } else {
      for (let message of this.messages) {
        message.isChecked = false;
        this.updateCheckedMessages(message);
      }
    }
  }

  sortDirection = 0;
  sortKey = null;

  sortBy(key) {
    this.sortKey = key;
    this.sortDirection++;
    this.sort();
  }

  sort() {
    if(this.sortDirection == 1) {
      this.filteredMessages = this.filteredMessages.sort((a,b) => {
        const valA = a[this.sortKey];
        const valB = b[this.sortKey];
        return valA.localeCompare(valB);
      });
    }
    else if(this.sortDirection == 2) {
      this.filteredMessages = this.filteredMessages.sort((a,b) => {
        const valA = a[this.sortKey];
        const valB = b[this.sortKey];
        return valB.localeCompare(valA);
      });
    } else {
      this.sortDirection = 0;
      this.sortKey = null;
    }
  }

}
