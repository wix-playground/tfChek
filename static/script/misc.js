
let last = null;
const urlParams = new URLSearchParams(location.search);

function start_scroll_down() {
    scroll = setInterval(function(){ window.scrollBy(0, 1000); console.log('start');}, 1500);
}

function stop_scroll_down() {
    clearInterval(scroll);
    console.log('stop');
}

function cancel() {
    let id = urlParams.get('id');
    let xmlHttp = new XMLHttpRequest();
    xmlHttp.onreadystatechange = function() {
        if (xmlHttp.readyState == 4 && xmlHttp.status == 202)
            console.log("Task "+id+" has been marked for deletion");
    }
    if (xmlHttp.readyState == 4 && xmlHttp.status >= 400){
        console.log("Failed to cancel task "+id);
    }
    xmlHttp.open("GET", "/api/v1/cancel/"+id, true); // true for asynchronous
    xmlHttp.send(null);
}

function copyText() {
    let text = "";
    let output = document.getElementById("output");
    let textArea = document.createElement("textarea");
    textArea.value = output.innerText;
    document.body.appendChild(textArea);
    textArea.focus();
    textArea.select();
    try {
        let successful = document.execCommand("copy");
        let message = successful ? 'successful' : 'unsuccessful';
        console.log("Copying was " + message);
    } catch (error) {
        console.error("Unable to copy", error);
    }
    //clearSelection()
    document.body.removeChild(textArea);
}

function clearSelection(){
    if (window.getSelection()) {
        window.getSelection().removeAllRanges();
    }  else if (document.getSelection()) {
        document.getSelection().empty();
    }
}

function writeToScreen(message) {
    let output = document.getElementById("output");
    let anchor = document.getElementById("anchor")
    let pre = document.createElement("pre");
    pre.style.wordWrap = "break-word";
    let msg = ansi_up.ansi_to_html(message)
    if (message.includes('\r')) {

        console.log("Rewriting: " + last.innerHTML)
        console.log("with: " + msg)
        last.innerHTML = msg

    } else {
        pre.innerHTML = msg;
        last = pre
        output.insertBefore(pre, anchor)
        window.scrollTo(0, output.scrollHeight)
    }
}

function connectToWebSocket() {

    console.log("Attempting Connection...");
    const ansi_up = new AnsiUp();
    if (urlParams.has('id')) {
        //Connect ot a websocket
        let proto = "ws";
        if (location.protocol === "https:") {
            proto = "wss"
        }
        let socket = new WebSocket(proto + "://" + location.hostname + ":" + location.port + location.pathname + "ws/runsh/" + urlParams.get('id'));
        socket.onopen = () => {
            console.log("Successfully Connected");
            socket.send("Hi From the Client!")
        };
        socket.onmessage = msg => {
            writeToScreen(msg.data)
        };
        socket.onclose = event => {
            console.log("Socket Closed Connection: ", event);
            socket.send("Client Closed!")
        };

        socket.onerror = error => {
            console.log("Socket Error: ", error);
            document.getElementById("messages").innerText = "Cannot connect to to websocket"
        };
    } else {
        document.getElementById("messages").innerText = "You have to query task by id number. (Add URL suffix like like '?id=1')"
    }
}

function controlScroll() {
    let bar = document.getElementById("bar");
    const sticky = bar.offsetTop;
    function stickTheBar() {
        if (window.pageYOffset >= sticky) {
            bar.classList.add("sticky");
        } else {
            bar.classList.remove("sticky");
        }
    }
    window.onscroll = function () {
        stickTheBar();
    }
}