import { Component, OnInit } from '@angular/core';

import { BrokerService } from '../broker.service';
import { Broker } from 'src/assets/model/broker';
import { Router } from '@angular/router';
import { Filter } from 'src/assets/model/filter';

@Component({
  selector: 'app-broker',
  templateUrl: './broker.component.html',
  styleUrls: ['./broker.component.scss'],
})
export class BrokerComponent implements OnInit {

  title = 'brokerUI';
  brokers: Broker[];
  selectedBroker: string;

  constructor(private brokerService: BrokerService, private router: Router) {
    this.brokers = [];
  }

  ngOnInit() {
    this.brokerService.getBrokers().subscribe(result => {
      this.brokers = result;
      this.brokerService.setBrokers(this.brokers);
    });
  }

  filterBroker(broker: Broker) {
    let filter = new Filter();
    filter.brokerName = broker.Name;
    this.brokerService.setFilter(filter);
    this.router.navigate(['brokers', filter.brokerName, 'queues']);
  }

}
