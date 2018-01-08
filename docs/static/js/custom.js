// Stick the top to the top of the screen when  scrolling
$("#top-bar").stick_in_parent({spacer: false});

jQuery(document).ready(function () {
    var sidebarActiveli = $('#sidebar').find('li.active').get(0);
    sidebarActiveli && sidebarActiveli.scrollIntoView();
});