SESSION="websockets"

# 3-way chat on the websocket server to more easily test interactions
CHAT_SERVER="ws://localhost:8081/ws"

tmux new-session -d -s $SESSION

tmux rename-window -t $SESSION:0 'player1'
tmux new-window -t $SESSION -n 'player2'
tmux new-window -t $SESSION -n 'player3'

tmux send-keys -t $SESSION:player1 "source ./functions.sh" Enter
tmux send-keys -t $SESSION:player2 "source ./functions.sh" Enter
tmux send-keys -t $SESSION:player3 "source ./functions.sh" Enter
tmux send-keys -t $SESSION:player1 "websocat $CHAT_SERVER" Enter
tmux send-keys -t $SESSION:player2 "websocat $CHAT_SERVER" Enter
tmux send-keys -t $SESSION:player3 "websocat $CHAT_SERVER" Enter
tmux send-keys -t $SESSION:player1 $(send-create-user 1 player1 "Player 1") Enter
tmux send-keys -t $SESSION:player2 $(send-create-user 2 player2 "Player 2") Enter
tmux send-keys -t $SESSION:player3 $(send-create-user 3 player3 "Player 3") Enter

tmux kill-window -t $SESSION:player1
tmux kill-window -t $SESSION:player2
tmux kill-window -t $SESSION:player3
tmux kill-session -t $SESSION