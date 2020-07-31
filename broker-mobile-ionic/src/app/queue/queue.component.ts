import { Component, OnInit } from '@angular/core';

import { BrokerService } from '../broker.service';
import { Queue } from 'src/assets/model/queue';
import { Filter } from 'src/assets/model/filter';
import { Router, Params, ActivatedRoute} from '@angular/router';
import { Broker } from 'src/assets/model/broker';
import { timeout } from 'rxjs/operators';

// Max time the client waits for queues in milliseconds
const getQueueTimeout = 60000;

@Component({
  selector: 'app-queue',
  templateUrl: './queue.component.html',
  styleUrls: ['./queue.component.scss'],
})
export class QueueComponent implements OnInit {
  title = 'brokerUI';
  queues: Queue[];
  brokers: Broker[];
  filter: Filter;
  isLoading = false;

  constructor(private brokerService: BrokerService, private router: Router, private route: ActivatedRoute) {
    this.queues = [];
    this.brokers = [];
  }

  

  ngOnInit() {
    this.filter = this.brokerService.getFilter();
    this.route.paramMap.subscribe(params => {
      this.filter.brokerName = params.get('broker');
    });

    if (this.filter.brokerName !== undefined) {
      this.getBrokers();
      this.getQueues();
    } else {
      this.brokerService.showToast('Please select a broker first', true);
      this.router.navigate(['./brokers']);
    }
  }

  getBrokers() {
    this.brokers = this.brokerService.getSubscribedBrokers();
    // If the user came through URL, ensure brokers are loaded
    if(this.brokers == null) {
      this.brokerService.getBrokers().subscribe(result => {
        this.brokers = result;
        this.brokerService.setBrokers(this.brokers);
      },
      error => {
        this.brokerService.showToast('Invalid URL, please ensure broker name is correct', true);
        this.router.navigate(['./brokers']);
      });
    }
  }
    

  getQueues() {
    this.brokerService.getQueues().pipe(timeout(getQueueTimeout)).subscribe(result => {
      this.isLoading = true;
      //this.brokerService.getMockQueues().subscribe(result => {
      this.queues = result;
      this.brokerService.setQueues(this.queues);
    }, (error) => {
      this.isLoading = true;
      this.brokerService.showToast(error, true);
      console.log('Error', error);
    });
    this.isLoading = false;
  }

  changeBroker() {
    this.brokerService.setFilter(this.filter);
    this.getQueues();
  }

  filterQueue(queue: Queue) {
    this.filter.queueName = queue.Name;
    this.brokerService.setFilter(this.filter);
    this.router.navigate(['brokers', this.filter.brokerName, 'queues', this.filter.queueName]);
  }

  purgeQueue(queueName: string) {
    let confirmed = confirm("Are you sure you want to purge " + queueName + " of ALL messages?");
    if (confirmed) {
      this.brokerService.deleteAllMessages(queueName).subscribe(result => {
        this.brokerService.showToast("Queue purged!", false);
        this.getQueues();
      });

    } else {
      console.log("Cancelled Purge");
    }
  }
}
