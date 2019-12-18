
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

    let xhttp = new XMLHttpRequest();
    const urlParams = new URLSearchParams(location.search);
    let from = window.location;
    if (urlParams.has("from")){
        from = urlParams.get("from");
    }
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4 && this.status == 200) {
            let providers =  JSON.parse(this.responseText)
        providers.forEach(function (value) {
            let setFunc = function(siteId){
                document.getElementById(containerId).append("<a href=\"/auth/" + value + "/login?site="+siteId+"&from="+from+"\">" +
                    "<img src=\"/static/pictures/"+value+".png\"/>" +
                    "</a>");
            };
            addProvider(value, setFunc);
        });
            let destination ="/auth/"+provider+"/login?site="+siteId+"&from="+location
            console.log("Redirecting to " + destination)
            window.location.replace(destination);
        }
    };
    xhttp.open("GET","/auth/list", true);
    xhttp.send();
}