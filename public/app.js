$(function() {
    $('form.deploy').submit(function() {
        $('iframe.console').show();
    })
    $('iframe.console').load(function() {
        console.log('hi')
    })
})