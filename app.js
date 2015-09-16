deploying = false
$(function() {
    $('button.deploy').click(function() {
        $.post($(this).attr("href"), function( data ) {
            console.log(data);
        });
        deploying = true
        poll()
        $(this).attr("disabled", true);
        return false;
    });
});
function longpoll(url, callback) {
    var req = new XMLHttpRequest();
    req.open('GET', url, true);
    req.responseType = "text";
    req.onerror = function(aEvt) {
        console.log("error");
    };
    req.onreadystatechange = function(aEvt) {
        if (req.readyState == 4) {
            if (req.status == 200) {
                var response = JSON.parse(req.responseText);
                writeToBody(response)
                if (response.IsReady === true && deploying === false) {
                    $('button.deploy').attr('disabled', false);
                }
                if (response.IsComplete !== true && deploying !== false) {
                    longpoll(url, callback);
                }
            } else {
                $('pre.console').append("\nError connecting to deployment server.");
            }
        }
    };

    req.send(null);
}

function writeToBody(responseJson) {
    $('pre.console').text(responseJson.Console);
}