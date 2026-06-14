import datetime

import websocket
import random
import json

messages = [
    "She loves reading books on sunny afternoons.",
    "The quick brown fox jumps over the lazy dog.",
    "Please remember to turn off the lights tonight.",
    "They decided to take a long vacation this summer.",
    "I need to buy some fresh groceries from the market.",
    "We will meet at the coffee shop around noon tomorrow.",
    "Learning a new language requires a lot of daily practice.",
    "The gentle breeze made the colorful autumn leaves dance gracefully.",
    "She carefully painted the beautiful landscape during her art class.",
    "He lost his car keys somewhere near the crowded parking lot.",
    "The little children played happily in the park until sunset approached.",
    "We are planning to visit our grandparents during the winter holidays.",
    "You should always double check your work before submitting it online.",
    "The bright stars illuminated the dark sky throughout the chilly night.",
    "They finally reached the top of the mountain after several hours. ",
]

CLIENTS = 100
ROUNDS = 1000

def on_message(wsapp, message):
    print(message)
def create_message(index, content):
    return {
        "type": "chat_message",
        "payload": {
            "userId": index,
            "channelId": 1,
            "content": content,
        }
    }

messages_by_client = {}
connections = []


# websocket.enableTrace(True)

for i in range(CLIENTS):
    ws = websocket.WebSocket()
    ws.connect("ws://localhost:8080/ws")
    connections.append(ws)
    messages_by_client[i] = {
        "sent": [],
        "received": [],
        "startedAt":  datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
    }

for round in range(ROUNDS):
    print(f"Starting round {round+1}")
    for i, ws in enumerate(connections):
        ws.ping("Is anyone home?")
        received = ws.recv()
        print(f"Client {i+1} received: {received}")
        messages_by_client[i]["received"].append(received)

        text = messages[random.randrange(0, len(messages))]
        msg = create_message(i+1, text)
        # print(f"Client {i+1} sending: {text}")
        ws.send(json.dumps(msg))
        messages_by_client[i]["sent"].append(text)
for i, ws in enumerate(connections):
    print(f"Closing connection {i+1}")
    messages_by_client[i]["closedAt"] = datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    ws.close(status=1000)

with open(f"chat-server-load-test-{datetime.datetime.now().strftime("%Y%m%d-%H%M%S")}.json", "w") as results:
    results.write(json.dumps(messages_by_client, indent=2))
