```bash
open "https://api.slack.com/"

# New webhook > Add Configuration

# Choose the channel

# Add Incomming WebHooks Integration

# Copy the Webhook URL

export URL=[...]

curl -X POST \
    --data 'payload={"channel": "#random", "username": "swarm-listener", "text": "This is posted to #random and comes from a bot named swarm-listener.", "icon_emoji": ":ghost:"}' \
    $URL
```