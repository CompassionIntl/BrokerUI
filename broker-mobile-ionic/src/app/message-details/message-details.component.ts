import { Component, OnInit, Input } from '@angular/core';
import { Message } from 'src/assets/model/message';
import { BrokerService } from '../broker.service';
import { ModalController } from '@ionic/angular';
import { Filter } from 'src/assets/model/filter';
import { Queue } from 'src/assets/model/queue';
import { Router, ActivatedRoute} from '@angular/router';

@Component({
  selector: 'app-message-details',
  templateUrl: './message-details.component.html',
  styleUrls: ['./message-details.component.scss'],
})
export class MessageDetailsComponent implements OnInit {
  @Input() message: Message;
  @Input() modalCtrl: ModalController;
  filter: Filter;
  queues: Queue[];
  deletedOrMoved = false;

  constructor(private brokerService: BrokerService, private router: Router, private route: ActivatedRoute) {
  }

  ngOnInit() {
    // this.message = this.brokerService.getSelectedMessage();
    this.queues = this.brokerService.getSubscribedQueues();
    this.filter = this.brokerService.getFilter();

  }

  changeDestinationQueue() {
    this.brokerService.setFilter(this.filter);
  }

  moveMessage() {
    if (this.filter.destinationQueueName === undefined || this.filter.destinationQueueName === this.filter.queueName) {
      this.brokerService.showToast("Please choose a queue to move message(s) to. \n Destination queue can not be the same as the current queue.", true);
    } else {
      let confirmed = confirm("Are you sure you want to move " + this.message.MessageID + " from: " + this.filter.queueName + " to: " + this.filter.destinationQueueName + "?");
      if (confirmed) {
        this.brokerService.moveMessage(this.message.MessageID).subscribe(result => {
          //TODO Check result and don't always show success toast
          console.log(result);
        });
        this.deletedOrMoved = true;
        this.brokerService.showToast("Message moved!", false);
        this.dismiss();
      } else {
        console.log("Cancelled Move");
      }
    }
  }

  deleteMessage() {
    let confirmed = confirm("Are you sure you want to delete " + this.message.MessageID + " ?");
    if (confirmed) {
      this.brokerService.deleteMessage(this.message.MessageID).subscribe(result => {
        //TODO Check result and don't always show success toast
        console.log(result);
      });
      this.deletedOrMoved = true;
      this.brokerService.showToast("Message deleted!", false);
      this.dismiss();
    } else {
      console.log("Cancelled Delete");
    }
  }

  dismiss() {
    // using the injected ModalController this page
    // can "dismiss" itself and optionally pass back data
    this.modalCtrl.dismiss({
      'dismissed': true,
      'deletedOrMoved': this.deletedOrMoved
    });
  }
}
