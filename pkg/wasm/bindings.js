/**
 * Fluxor EventBus WASM Client Bindings
 * 
 * This file provides a JavaScript wrapper for the WASM EventBus client.
 * It handles WebSocket connection and message routing.
 */

class FluxorEventBusClient {
    constructor(wsURL) {
        this.wsURL = wsURL;
        this.ws = null;
        this.connected = false;
        this.messageHandlers = new Map();
        this.pendingRequests = new Map();
    }

    /**
     * Connect to EventBus via WebSocket
     */
    async connect() {
        return new Promise((resolve, reject) => {
            try {
                // Connect WebSocket
                const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
                const wsURL = this.wsURL.startsWith('ws://') || this.wsURL.startsWith('wss://') 
                    ? this.wsURL 
                    : `${protocol}//${window.location.host}${this.wsURL}`;

                this.ws = new WebSocket(wsURL);

                this.ws.onopen = () => {
                    this.connected = true;
                    console.log('EventBus WebSocket connected');
                    resolve();
                };

                this.ws.onmessage = (event) => {
                    const msg = JSON.parse(event.data);
                    this.handleMessage(msg);
                };

                this.ws.onerror = (error) => {
                    console.error('WebSocket error:', error);
                    reject(error);
                };

                this.ws.onclose = () => {
                    this.connected = false;
                    console.log('EventBus WebSocket closed');
                };
            } catch (error) {
                reject(error);
            }
        });
    }

    /**
     * Handle incoming WebSocket messages
     */
    handleMessage(msg) {
        // Handle replies to requests
        if (msg.id && this.pendingRequests.has(msg.id)) {
            const { resolve, reject } = this.pendingRequests.get(msg.id);
            this.pendingRequests.delete(msg.id);

            if (msg.error) {
                reject(new Error(msg.error));
            } else {
                resolve(msg.result);
            }
            return;
        }

        // Handle subscription messages
        if (msg.op === 'message' && msg.address) {
            const handlers = this.messageHandlers.get(msg.address);
            if (handlers) {
                handlers.forEach(handler => {
                    try {
                        handler({
                            body: msg.body,
                            headers: msg.headers || {},
                            address: msg.address
                        });
                    } catch (error) {
                        console.error('Error in message handler:', error);
                    }
                });
            }
        }
    }

    /**
     * Send message via WebSocket
     */
    sendMessage(msg) {
        if (!this.connected || !this.ws) {
            throw new Error('Not connected');
        }

        this.ws.send(JSON.stringify(msg));
    }

    /**
     * Publish a message
     */
    publish(address, body) {
        const msg = {
            op: 'publish',
            address: address,
            body: body
        };

        this.sendMessage(msg);
    }

    /**
     * Send a point-to-point message
     */
    send(address, body) {
        const msg = {
            op: 'send',
            address: address,
            body: body
        };

        this.sendMessage(msg);
    }

    /**
     * Send a request and wait for reply
     */
    request(address, body, timeout = 5000) {
        return new Promise((resolve, reject) => {
            const requestID = `req-${Date.now()}-${Math.random()}`;

            const msg = {
                op: 'request',
                address: address,
                body: body,
                id: requestID,
                timeout: timeout
            };

            // Store pending request
            this.pendingRequests.set(requestID, { resolve, reject });

            // Set timeout
            setTimeout(() => {
                if (this.pendingRequests.has(requestID)) {
                    this.pendingRequests.delete(requestID);
                    reject(new Error('Request timeout'));
                }
            }, timeout);

            this.sendMessage(msg);
        });
    }

    /**
     * Subscribe to an address
     */
    subscribe(address, handler) {
        if (!this.messageHandlers.has(address)) {
            this.messageHandlers.set(address, []);
        }

        this.messageHandlers.get(address).push(handler);

        // Send subscribe message
        const msg = {
            op: 'subscribe',
            address: address
        };

        this.sendMessage(msg);

        // Return unsubscribe function
        return () => {
            this.unsubscribe(address, handler);
        };
    }

    /**
     * Unsubscribe from an address
     */
    unsubscribe(address, handler) {
        const handlers = this.messageHandlers.get(address);
        if (handlers) {
            const index = handlers.indexOf(handler);
            if (index > -1) {
                handlers.splice(index, 1);
            }

            // If no more handlers, send unsubscribe
            if (handlers.length === 0) {
                this.messageHandlers.delete(address);
                const msg = {
                    op: 'unsubscribe',
                    address: address
                };
                this.sendMessage(msg);
            }
        }
    }

    /**
     * Close connection
     */
    close() {
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
        this.connected = false;
        this.messageHandlers.clear();
        this.pendingRequests.clear();
    }
}

// Export for use in browser
if (typeof window !== 'undefined') {
    window.FluxorEventBusClient = FluxorEventBusClient;
}

// Export for Node.js
if (typeof module !== 'undefined' && module.exports) {
    module.exports = FluxorEventBusClient;
}

