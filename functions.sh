function send-create-user() {
    MSG="{""type"": ""create_user"", ""payload"":{id: ""$1"", username: ""$2"", displayName: ""$3""}}"
    echo $MSG
}