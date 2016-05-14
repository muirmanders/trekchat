(function() {

  var ws;

  var broadcast_message = function(message) {
    ws.send(JSON.stringify({
      command: "send_message",
      args: {
        message: message
      }
    }));
  };

  var private_message = function(recipient, message) {
    ws.send(JSON.stringify({
      command: "send_message",
      args: {
        message: message,
        recipient: recipient,
        private: true
      }
    }));
  };

  var open_websocket = function() {
    if (location.protocol == "https:") {
      ws = this.ws = new WebSocket("wss://" + location.host + "/connect");
    } else {
      ws = this.ws = new WebSocket("ws://" + location.host + "/connect");
    }

    var chat_frame = $("#container .chat-frame");

    ws.onmessage = function(event) {
      var cmd = JSON.parse(event.data);
      switch (cmd.command) {
      case "message":
        var msg = $("<p>").
          addClass("chat-message").
          text(cmd.args.sender + ": " + cmd.args.message);
        if (cmd.args.private) {
          msg.addClass("private");
        } else if (cmd.args.from_me) {
          msg.addClass("from-me");
        }
        var was_at_bottom = chat_frame.prop("scrollHeight") - chat_frame.scrollTop() == chat_frame.height();
        chat_frame.append(msg);
        if (was_at_bottom) {
          chat_frame.animate({scrollTop: chat_frame.prop("scrollHeight")}, 200);
        }
        break;
      case "error":
        var msg = $("<p>");
        msg.text(cmd.args.message);
        msg.addClass("chat-message");
        msg.addClass("error");
        chat_frame.append(msg);
        break;
      case "welcome":
        var msg = $("<p>");
        msg.text("Welcome! You are " + cmd.args.name);
        msg.addClass("chat-message");
        msg.addClass("welcome");
        chat_frame.append(msg);
        break;
      case "users":
        var users_frame = $("#container .users-frame");
        users_frame.empty();
        for (var i = 0; i < cmd.args.users.length; i++) {
          users_frame.append($("<p>").text(cmd.args.users[i]));
        }
        break;
      }
    };

    ws.onopen = function() {
      $("#container").show();
    };

    ws.onerror = function(e) {
      console.log("websocket error: " + e);
    };
  };

  $(open_websocket);

  $(function() {
    $("input").keydown(function(e) {
      if (e.keyCode == 13) {
        var msg = $(this).val();
        if (msg.length == 0) {
          return;
        }

        if (msg[0] == "/") {
          var match = msg.match(/^\/dm\s+(\S+)\s+(.+)$/);
          if (match) {
            private_message(match[1], match[2]);
          } else {
            return;
          }
        } else {
          broadcast_message(msg);
        }

        $(this).val("");
      }
    });
  });
})();
