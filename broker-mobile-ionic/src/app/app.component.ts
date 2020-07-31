import { Component, OnInit } from '@angular/core';

import { Router, NavigationEnd } from '@angular/router';

import { Broker } from 'src/assets/model/broker';
import { BrokerService } from './broker.service';

import { Platform } from '@ionic/angular';
import { Filter } from 'src/assets/model/filter';

@Component({
  selector: 'app-root',
  templateUrl: 'app.component.html',
  styleUrls: ['app.component.scss']
})
export class AppComponent implements OnInit {
  title = 'brokerUI';
  brokers: Broker[];
  selectedTab: number;
  brokerSelected: boolean;
  queueSelected: boolean;
  currentFilter: Filter;

  brokerName: string;
  queueName: string;

  constructor(
    private platform: Platform,
    private brokerService: BrokerService,
    private router: Router
  ) {
    this.brokers = [];
    this.brokerSelected = false;
    this.queueSelected = false;
    // this.initializeApp();
  }

  ngOnInit() {
    // Use matchMedia to check the user preference
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)');

    toggleDarkTheme(prefersDark.matches);
    
    // Listen for changes to the prefers-color-scheme media query
    prefersDark.addListener((mediaQuery) => toggleDarkTheme(mediaQuery.matches));
    
    // Add or remove the "dark" class based on if the media query matches
    function toggleDarkTheme(shouldAdd) {
      document.body.classList.toggle('dark', shouldAdd);
    }

    this.router.events.subscribe(event => {
      if (event instanceof NavigationEnd) {

        // Decompose router url to get broker and queue name
        var urlParts = this.router.url.split('/');
        for (var i = 0; i < urlParts.length; i++) {
          if(urlParts[i] === 'brokers' && i+1 < urlParts.length)
            this.brokerName = urlParts[i+1];
          else if(urlParts[i] === 'queues' && i+1 < urlParts.length)
            this.queueName = urlParts[i+1];
        }

        this.currentFilter = this.brokerService.getFilter();
        this.currentFilter.brokerName = this.brokerName;
        this.currentFilter.queueName = this.queueName;

        if(this.router.url.includes('/message-details') ||
          this.router.url.includes('/messages') ||
          this.router.url.match(/\/queues\/+/)) {
          this.selectedTab = 3;
          this.brokerSelected = true;
          this.queueSelected = true;
        }
        else if(this.router.url.includes('/queues') ||
          this.router.url.match(/\/brokers\/+/)) {
          this.selectedTab = 2;
          this.brokerSelected = true;
          this.queueSelected = false;
        }
        else if(this.router.url.includes('/brokers')) {
          this.selectedTab = 1;
          this.brokerSelected = false;
          this.queueSelected = false;
        }
      }
    },
      error => {
        console.log(error);
      });
    this.selectedTab = 1;
  }

  // initializeApp() {

  // }

  w3_open() {
    document.getElementById("mySidebar").style.display = "block";
    document.getElementById("myOverlay").style.display = "block";
  }

  w3_close() {
    document.getElementById("mySidebar").style.display = "none";
    document.getElementById("myOverlay").style.display = "none";
  }

  selectTab(tabNum: number) {
    this.selectedTab = tabNum;
  }
}
