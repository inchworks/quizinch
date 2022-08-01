// Quiz controller extension to deck.js. Based on deck.menu.js

(function ($, undefined) {
  var $document = $(document);
  var $html = $('html');
  var rootSlides;

  /*
  Add the methods and key binding to quit a slide page.
  */

  var bindKeyEvents = function () {
    var options = $.deck('getOptions');
    $document.unbind('keydown.deckquiz');
    $document.bind('keydown.deckquiz', function (event) {
      var isQuizKey = event.which === options.keys.quiz;
      isQuizKey = isQuizKey || $.inArray(event.which, options.keys.quiz) > -1;
      if (isQuizKey && !event.ctrlKey) {
        window.location.href = (gblPuppet == "C") ? "/controller" : "/displays";
        event.preventDefault();
      }
    });
  };

  /*
  Extends defaults/options.

  options.keys.menu
    The numeric keycodes used to quit the slideshow.

  options.touch.doubletapWindow
    Not used, but referenced in a test.
  */
  $.extend(true, $.deck.defaults, {

    keys: {
      // esc, q, x
      quiz: [27, 81, 90] // ## deck.menu had a single value, no array
    },

    touch: {
      doubletapWindow: 400
    }
  });


  // Bind extension to core 

  $document.bind('deck.init', function () {
    bindKeyEvents();
  });

  /*
  Bind extension to auto-play audio and video on slides.
  */

  $(document).bind('deck.change', function (event, from, to) {

    // pause previous audio or slide, and play current
    let $fromSlide = $.deck('getSlide', from);
    $fromSlide.find('.playable').each(function (i, av) {
      if (av.currentTime > 0 && av.readyState > 2)
        av.pause();
    });

    // play current slide
    let $toSlide = $.deck('getSlide', to);
    if ($toSlide.hasClass('av')) {
      $toSlide.find('.playable').each(function (i, av) {
        av.play();
      });
    }

  });

})(jQuery);
