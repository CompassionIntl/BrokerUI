import { NgModule } from '@angular/core';
import { PreloadAllModules, RouterModule, Routes } from '@angular/router';

import { QueueComponent } from './queue/queue.component';
import { MessageComponent } from './message/message.component';
import { BrokerComponent } from './broker/broker.component';
import { MessageDetailsComponent } from './message-details/message-details.component';


const routes: Routes = [
  { path: 'brokers', component: BrokerComponent },
  { path: 'brokers/:broker', component: QueueComponent },
  { path: 'brokers/:broker/queues', component: QueueComponent },
  { path: 'brokers/:broker/queues/:queue', component: MessageComponent },
  { path: 'brokers/:broker/queues/:queue/messages', component: MessageComponent },
  { path: 'brokers/:broker/queues/:queue/messages/message-details', canActivate: [false], component: MessageDetailsComponent },
  { path: '', redirectTo: '/brokers', pathMatch: 'full' },
  { path: '**', redirectTo: '/brokers' },
];

@NgModule({
  imports: [
    RouterModule.forRoot(routes, { preloadingStrategy: PreloadAllModules })
  ],
  exports: [RouterModule]
})
export class AppRoutingModule {}
