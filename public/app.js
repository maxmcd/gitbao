$(function() {
    var success = false;
    $('form.deploy').submit(function() {
        $('iframe.console').show();
        $(this).find("input[type='submit']").val('Deploying...')
    })
    $('iframe.console').load(function() {
        // Do something?
    })
    window.addEventListener("message", function(event) {
        success = true
        url = "http://" + event.data
        $('.setup').hide()
        $('.complete a').attr('href', url)
        $('.complete .textlink').text(url)
        $('.complete').show()
    }, false);
})