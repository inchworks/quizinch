// Copyright © Rob Burke inchworks.com, 2020.
//
// Monitor display functions.

var $table;
var $prototype;
var $tRep;

jQuery(document).ready(function() {

     // get the table that holds the collection of items
    $table = $('#monitor');

    // live check
    $live = $('#liveW');
    $tRep = Date.now();
});

// Add display as a row in table

function addDisplay($t, d) {

    // row
    var $r = $("<tr>");

    // set display name in first td
    var $td = $("<td>").attr("class", "clientName").text(d.Name);
    $r.append($td);
 
    // append periods in td
    d.Periods.forEach(function(p) {
        total = p.Lost + p.Missed;
        $td = $("<td>").attr("class", "period" + p.Status).text(total + " : " + p.Longest);
        $r.append($td);
    });

    // add row to table
    $t.append($r);
}

// Show monitor is live

function monitorLive() {

    // seconds since last response
    w = Math.round((Date.now() - $tRep)/1000);
    if (w < 7)
        st = "G";
    else if (w < 12)
        st = "A";
    else
        st = "R";

    $live.text(w);
    $live.attr("class", "period" + st);
}

// Refresh display

function monitorUpdate() {

	// get current status
	$.post(
		'/monitor-update',
		{ csrf_token: gblToken }, 
		function(response){

               // time of response
                $tRep = Date.now();

                // get the JSON collection of items
                var items = response.Displays;

                // remove existing rows
                $table.empty();

                // add row for each item (may be null if no live displays)
                if (items !== null) { 
                    items.forEach(function(item) {
                        addDisplay($table, item);
                    });
                }               
		    },
		'json');
}

setInterval(monitorUpdate, gblInterval);
setInterval(monitorLive, 1000);
