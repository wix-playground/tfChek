const xhttp = new XMLHttpRequest();

function addProvider(provider, setter) {
    xhttp.onreadystatechange = function () {
        if (this.readyState == 4 && this.status == 200) {
            let siteId = this.responseText.trim().toLowerCase();
            setter(siteId);
        }
    };
    xhttp.open("GET", "authinfo/" + provider, true);
    xhttp.send();
}

function setProviders(containerId) {
    let urlParams = new URLSearchParams(location.search);
    let from = window.location;
    if (urlParams.has("from")) {
        from = urlParams.get("from");
    }
    xhttp.onreadystatechange = function () {
        if (this.readyState == 4 && this.status == 200) {
            let providers = JSON.parse(this.responseText)
            providers.forEach(function (value) {
                let setFunc = function (siteId) {
                    let pl = document.createElement("a");
                    pl.setAttribute("href", "auth/" + value + "/login?site=" + siteId + "&from=" + from);
                    let pi = document.createElement("img");
                    pi.setAttribute("src", "static/pictures/" + value + ".png");
                    pl.appendChild(pi);
                    document.getElementById(containerId).appendChild(pl);
                };
                addProvider(value, setFunc);
            });
        }
    };
    xhttp.open("GET", "auth/list", true);
    xhttp.send();
}

function checkAuthentication() {
    let xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function () {
        if (this.readyState == 4) {
            if (this.status >= 400 || this.responseText.replace(/(\r\n|\n|\r)/gm, "") == "null") {
                window.location.replace("login?from=" + window.location);
            }
            if (this.status == 200) {
                let logoutButton = document.createElement("a");
                logoutButton.setAttribute("id", "logout");
                logoutButton.setAttribute("onclick", "logout()");
                logoutButton.setAttribute("href", "javascript:void(0)");
                logoutButton.appendChild(document.createTextNode("Logout"));
                document.getElementById("bar").appendChild(logoutButton);
            }
        }
    };
    xhttp.open("GET", "auth/user", true);
    xhttp.send();
}

function getUserInfo(callback) {
    let info = null;
    let xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function () {
        if (this.readyState == 4 && this.status == 200) {
            info = JSON.parse(this.responseText);
            callback(info)
        }
    };
    xhttp.open("GET", "auth/user", true);
    xhttp.send();
}

function logout() {
    let xhttp = new XMLHttpRequest();
    getUserInfo(function (userInfo) {
        let p = userInfo.id.split("_")[0];
        xhttp.onreadystatechange = function () {
            if (this.readyState == 4 && this.status == 200) {
                console.log("You have been logged out!")
            }
        };
        xhttp.open("GET", "auth/" + p + "/logout", true);
        xhttp.send();
    });
}