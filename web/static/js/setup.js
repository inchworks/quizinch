// Copyright © Rob Burke inchworks.com, 2020.

// Client-side functions.

// Add and remove sub-forms on the Quiz and Round setup pages.
// Based on Symfony Cookbook "How to Embed a Collection of Forms"
// I have attempted to generalise the code as much as possible. It assumes only one set of sub-forms per page. 

var $collectionHolder;
var $prototype;

// setup an "add entity" link
// ## Could put button text as global in page template to make it specific
var $addChildLink = $('<a href="#" class="btnAddChild button">Add</a>');
var $newLinkLi = $('<li></li>').append($addChildLink);

jQuery(document).ready(function() {

     // Get the ul that holds the collection of items
    $collectionHolder = $('#formChildren');

    // prototype sub-form is the first one
    $prototype = $collectionHolder.find('li').first();

    // add the "add a child" anchor and li to the tags ul
    $collectionHolder.append($newLinkLi);

    // get the JSON collection of items
    var items = JSON.parse( $('#items').val() );

     // add sub-form for each item
    items.forEach(function(item) {
        var newForm = addChildForm($collectionHolder, $newLinkLi);
        newForm.find('div').data('index', item.Index);
        newForm.find(".error").text(item.Error);

        // page-specific function
        setItemFields(newForm, item);
    });
    
    // add a delete link to all of the existing child form li elements
    // $collectionHolder.find('li').each(function() {
    //    addChildFormDeleteLink($(this));
    // });

 
    // count the current form inputs we have (e.g. 2), use that as the new
    // index when inserting a new item (e.g. 2)
    $collectionHolder.data('index', $collectionHolder.find(':input').length);

    $addChildLink.on('click', function(e) {
        // prevent the link from creating a "#" on the URL
        e.preventDefault();

        // add a new child form (see next code block)
        addChildForm($collectionHolder, $newLinkLi);
    });
    
    // save items on form submit
    $('#submit').on('click', function(e) {
        saveChildFormData($collectionHolder);
    });


	// hide the spurious div that appears when the collection is empty
	// (Symfony bug, I think)
	// ## Needs code per-page if still a problem
	// $('div#quiz_teams').parent().hide();
});

function addChildFormDeleteLink($itemForm) {
    var $removeFormA = $('<a href="#" class="button btnDeleteChild">X</a>');
    $itemForm.find('input').first().after($removeFormA);

    $removeFormA.on('click', function(e) {
        // prevent the link from creating a "#" on the URL
        e.preventDefault();

        // remove the li for the deleted item
        $itemForm.remove();
    });
}

function addChildForm($collectionHolder, $newLinkLi) {

    // clone the prototype
    var $newForm = $prototype.clone();

    // set the new index (and increment it)
    var index = $collectionHolder.data('index');
    $collectionHolder.data('index', index + 1);
    $newForm.find('div').first().data('index', index);

    // make form visible
    $newForm.css('display', 'block');
 
     // display the form in the page in an li, before the "Add" link li
     $newLinkLi.before($newForm);
	
	// add a delete link to the new form
    addChildFormDeleteLink($newForm);

    return $newForm;
}

function saveChildFormData($collectionHolder) {

    var items = [];

    $collectionHolder.find('li').each(function() {
    
        var index = $(this).find('div').first().data('index');

        if (index >= 0) {

            // page-specific function
            var item = getItemFields($(this));

            // add original data index
            item.Index = $(this).find('div').first().data('index');

            items.push(item);
        }
    });

    // return in hidden field
    $('#items').val(JSON.stringify(items));
}