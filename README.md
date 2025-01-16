# Distributed Low-Latency Chat Server

A dynamic, distributed chat server built using Mininet, designed for low latency and high reliability. This project implements optimal server selection, load balancing, traffic congestion control, and secure communication to enhance the chat experience in distributed network environments.

## Demo
[Demo video on Youtube](https://www.youtube.com/watch?v=6KxOdmUmIco)

## Features

- **Dynamic Server Selection:** Implements an algorithm to connect clients to the most optimal server based on real-time network conditions, reducing average chat latency from 12 seconds to 3 seconds.
- **Load Balancing:** Balances client requests across multiple servers to prevent overloading and ensure even distribution of traffic.
- **Traffic Congestion Control:** Monitors network performance to adjust traffic flow dynamically, avoiding bottlenecks and improving reliability.
- **Secure Communication:** Utilizes TLS to ensure end-to-end encryption for all messages, safeguarding user data and communication.

## Architecture

The chat server uses a distributed architecture where multiple servers collaborate to provide seamless communication. Clients connect to the most suitable server based on network metrics, ensuring low latency and efficient resource usage.

## Paper
This project was completed as our final project for Computer Networks (CSCD58) at UofT. The report/motivation for this project can be seen in report.pdf.
