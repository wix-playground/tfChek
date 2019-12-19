const xhttp = new XMLHttpRequest();
function addProvider(provider, setter) {
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4 && this.status == 200) {
            let siteId = this.responseText.trim().toLowerCase();
            setter(siteId);
        }
    };
    xhttp.open("GET","/authinfo/"+provider, true);
    xhttp.send();
}
function setProviders(containerId){
    let urlParams = new URLSearchParams(location.search);
    let from = window.location;
    if (urlParams.has("from")){
        from = urlParams.get("from");
    }
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4 && this.status == 200) {
            let providers =  JSON.parse(this.responseText)
        providers.forEach(function (value) {
            let setFunc = function(siteId){
                let pl = document.createElement("a");
                pl.setAttribute("href", "/auth/" + value + "/login?site="+siteId+"&from="+from);
                let pi = document.createElement("img");
                pi.setAttribute("src","static/pictures/"+value+".png");
                pl.appendChild(pi);
                document.getElementById(containerId).appendChild(pl);
            };
            addProvider(value, setFunc);
        });
        }
    };
    xhttp.open("GET","/auth/list", true);
    xhttp.send();
}