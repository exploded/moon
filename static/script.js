var marker;
var myMap;
var mylat = -37;
var mylon = 144;
var myzon = 10

function setupMap() {
	getLocationFromBrowser();
	//
	showGoogleMaps();


};

function getLocationFromBrowser() {
	// Get golocation from Browser
	var x = document.getElementById("errormessage");
    if (navigator.geolocation) {
        navigator.geolocation.getCurrentPosition(showgeolocation, showError);		
    } else { 
        x.innerHTML = "Geolocation is not supported by this browser.";
    }
}

function showgeolocation(position) {
	//Display lat and lon
	mylat=parseFloat(position.coords.latitude).toFixed(4);
	mylon=parseFloat(position.coords.longitude).toFixed(4);	
	$("#Latitude").get(0).MaterialTextfield.change(mylat);
	$("#Longitude").get(0).MaterialTextfield.change(mylon);
	moveMarker()
	updateCalLink()
	//
	displayTimeZone();
	getTimes();

};

function moveMarker()
{
	var myLatLon = new google.maps.LatLng( mylat, mylon )
	marker.setVisible(false);
	myMap.panTo( myLatLon );
	marker.setPosition( myLatLon );
	marker.setVisible(true);
}

function refresh() {
	getLocationFromBrowser();
	//changeSpinners();
};

function displayTimeZone() {
	var d = new Date()
	var n = -d.getTimezoneOffset()/60;	
	$("#Timezone").get(0).MaterialTextfield.change(n);	
};

function showGoogleMaps(){
    var latlon = new google.maps.LatLng(mylat, mylon)
    mapholder = document.getElementById('mapholder')
    mapholder.style.height = '250px';
    mapholder.style.width = '512px';
    var myOptions = {
	    center:latlon,zoom:8,
	    mapTypeId:google.maps.MapTypeId.ROADMAP,
	    mapTypeControl:false,
	    navigationControlOptions:{style:google.maps.NavigationControlStyle.SMALL}
    }    
    myMap = new google.maps.Map(document.getElementById("mapholder"), myOptions);
    marker = new google.maps.Marker({position:latlon,draggable: true,map:myMap,title:"Move marker to your location."});	
	addListener(marker);
};

// Detect when the marker is moved or dragged by the user
function addListener(marker)
{
 google.maps.event.addListener(marker, 'dragend', function(event) {
		mylat=parseFloat(event.latLng.lat()).toFixed(4);
		mylon=parseFloat(event.latLng.lng()).toFixed(4);	
		$("#Latitude").get(0).MaterialTextfield.change(mylat);
		$("#Longitude").get(0).MaterialTextfield.change(mylon);
		getTimes()
		updateCalLink()
});
}

// Detect when user changed lat or long input, and move the marker in the map
function SpinnersChanged()
{
	mylat=$('#lat').val();
	mylon=$('#lon').val();
	var myLatLon = new google.maps.LatLng( mylat, mylon )
	moveMarker()
	getTimes();
	 updateCalLink()
}



// Change the calendar link after a lat / on change
function updateCalLink()
{	
    document.getElementById("callink").href="calendar?lat="+mylat+"&lon="+mylon+"&zon="+myzon;     
}

// Change the values in the spinners after a map change
function updateSpinners()
{	
	$("#lat").get(0).MaterialTextfield.change(mylat);
	$("#lon").get(0).MaterialTextfield.change(mylon);	
	var myLatLon = new google.maps.LatLng( mylat, mylon )
	moveMarker()
	updateCalLink()
}

// Get the rise and set times from the server
var getTimes = function(){	
	var zon = $('#zon').val();	
	myzon=zon;
	$.getJSON("gettimes?lon=" + mylon + "&lat="+mylat+"&zon="+zon, function(json) {			
			$("#Rise").get(0).MaterialTextfield.change(json.Rise);
			$("#Set").get(0).MaterialTextfield.change(json.Set);	
	});
};

function showError(error) {
	var x = document.getElementById("errormessage");	
    switch(error.code) {
        case error.PERMISSION_DENIED:
            x.innerHTML = "User denied the request for Geolocation."
            break;
        case error.POSITION_UNAVAILABLE:
            x.innerHTML = "Location information is unavailable."
            break;
        case error.TIMEOUT:
            x.innerHTML = "The request to get user location timed out."
            break;
        case error.UNKNOWN_ERROR:
            x.innerHTML = "An unknown error occurred."
            break;
    }
}