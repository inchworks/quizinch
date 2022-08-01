// Copyright © Rob Burke inchworks.com, 2020.

// Control-puppet functions for use with Deck.js.

// Send controller action

function controlChange(newIndex) {

	gblIndex = newIndex;

	// report current position (ignoring response), and if touchscreen navigation is needed
	if (gblPuppet == "C")
		$.post('/control-change', { index: newIndex, touchNav: gblTouchNav, csrf_token: gblToken });
}

// Get new position

function controlPuppet() {

	// ## can I pick up deck.js currentIndex somehow, instead of my own copy?  

	// report current position and get new position
	$.post(
		'/control-puppet/',
		{ puppet: gblPuppet, access: gblAccess, page: gblPage, param: gblParam, index: gblIndex, update: gblUpdate, monitor: gblMonitor, csrf_token: gblToken },
		function (response) {

			// check if new page (or refresh) required
			if (response.newHRef != '')
				window.location.href = response.newHRef;

			else {

				// update tick text on page
				if (response.newTick != '') {
					gblTick = response.newTick;
					$(".tick").text(response.newTick);
				}

				// check if slide changed
				if (response.newIndex != gblIndex)
					$.deck('go', response.newIndex);
			}
		},
		'json');
}

// Get next, or previous, set of slides

function controlStep(next) {

	$.post(
		'/control-step',
		{ next: next, csrf_token: gblToken },
		function (response) {
			window.location.href = response.newHRef;
		},
		'json');
}

// Refresh controller when scores become available or change

function controlUpdate() {

	// count the seconds
	++gblSecond;

	// check if refresh needed
	$.post(
		'/control-update',
		{ page: gblPage, access: gblAccess, param: gblParam, index: gblIndex, update: gblUpdate, second: gblSecond, monitor: gblMonitor, csrf_token: gblToken },
		function (response) {

			// check if new page or refresh required
			if (response.newHRef != '')
				window.location.href = response.newHRef;

			else if (response.newTick != '') {

				// update tick text on page
				gblTick = response.newTick;
				$(".tick").text(response.newTick);
			}
		},
		'json');
}

// Initialise slides

$(function () {
	$.deck('.slide');

	// Go to specified slide.
	// (Needed for controller after a scores update. Might also avoid a stutter on puppets.)
	if (gblIndex != 0)
		$.deck('go', gblIndex);
});


// Timer to check puppet position, or refresh content of controller slide

if (gblPuppet != "C")
	setInterval(controlPuppet, gblInterval);

else
	setInterval(controlUpdate, gblInterval);
