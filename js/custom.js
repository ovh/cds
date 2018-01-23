// Stick the top to the top of the screen when  scrolling
$("#top-bar").stick_in_parent({spacer: false});

function GetLatestReleaseInfo() {
    $.getJSON("https://api.github.com/repos/ovh/cds/releases/latest").done(function(release) {
      for (var i = 0; i < release.assets.length; i++) {
        $("."+release.assets[i].name).text(release.assets[i].name);
        $("."+release.assets[i].name).fadeIn("slow");
        $(".download-"+release.assets[i].name).attr("href", release.assets[i].browser_download_url);
      }
    });
  }

jQuery(document).ready(function () {
    var sidebarActiveli = $('#sidebar').find('li.active').get(0);
    sidebarActiveli && sidebarActiveli.scrollIntoView();
    GetLatestReleaseInfo();
});