'use strict';

let marker;
let myMap;
let mylat = -37;
let mylon = 144;
let myzon = 10;

// Comprehensive timezone list with UTC offsets
const timezones = [
	{ name: "Pacific/Midway", label: "(UTC-11:00) Midway Island, American Samoa", offset: -11 },
	{ name: "Pacific/Honolulu", label: "(UTC-10:00) Hawaii", offset: -10 },
	{ name: "America/Anchorage", label: "(UTC-09:00) Alaska", offset: -9 },
	{ name: "America/Los_Angeles", label: "(UTC-08:00) Pacific Time (US & Canada)", offset: -8 },
	{ name: "America/Tijuana", label: "(UTC-08:00) Tijuana, Baja California", offset: -8 },
	{ name: "America/Denver", label: "(UTC-07:00) Mountain Time (US & Canada)", offset: -7 },
	{ name: "America/Phoenix", label: "(UTC-07:00) Arizona", offset: -7 },
	{ name: "America/Chihuahua", label: "(UTC-07:00) Chihuahua, La Paz, Mazatlan", offset: -7 },
	{ name: "America/Chicago", label: "(UTC-06:00) Central Time (US & Canada)", offset: -6 },
	{ name: "America/Mexico_City", label: "(UTC-06:00) Mexico City", offset: -6 },
	{ name: "America/Regina", label: "(UTC-06:00) Saskatchewan", offset: -6 },
	{ name: "America/New_York", label: "(UTC-05:00) Eastern Time (US & Canada)", offset: -5 },
	{ name: "America/Bogota", label: "(UTC-05:00) Bogota, Lima, Quito", offset: -5 },
	{ name: "America/Caracas", label: "(UTC-04:00) Caracas", offset: -4 },
	{ name: "America/Halifax", label: "(UTC-04:00) Atlantic Time (Canada)", offset: -4 },
	{ name: "America/Santiago", label: "(UTC-04:00) Santiago", offset: -4 },
	{ name: "America/St_Johns", label: "(UTC-03:30) Newfoundland", offset: -3.5 },
	{ name: "America/Sao_Paulo", label: "(UTC-03:00) Brasilia", offset: -3 },
	{ name: "America/Argentina/Buenos_Aires", label: "(UTC-03:00) Buenos Aires", offset: -3 },
	{ name: "America/Godthab", label: "(UTC-03:00) Greenland", offset: -3 },
	{ name: "Atlantic/South_Georgia", label: "(UTC-02:00) Mid-Atlantic", offset: -2 },
	{ name: "Atlantic/Azores", label: "(UTC-01:00) Azores", offset: -1 },
	{ name: "Atlantic/Cape_Verde", label: "(UTC-01:00) Cape Verde Islands", offset: -1 },
	{ name: "Europe/London", label: "(UTC+00:00) London, Dublin, Lisbon", offset: 0 },
	{ name: "Europe/Paris", label: "(UTC+01:00) Paris, Brussels, Madrid", offset: 1 },
	{ name: "Europe/Berlin", label: "(UTC+01:00) Berlin, Rome, Amsterdam", offset: 1 },
	{ name: "Africa/Lagos", label: "(UTC+01:00) West Central Africa", offset: 1 },
	{ name: "Europe/Athens", label: "(UTC+02:00) Athens, Istanbul, Helsinki", offset: 2 },
	{ name: "Africa/Cairo", label: "(UTC+02:00) Cairo", offset: 2 },
	{ name: "Africa/Johannesburg", label: "(UTC+02:00) Johannesburg, Pretoria", offset: 2 },
	{ name: "Europe/Moscow", label: "(UTC+03:00) Moscow, St. Petersburg", offset: 3 },
	{ name: "Asia/Baghdad", label: "(UTC+03:00) Baghdad", offset: 3 },
	{ name: "Asia/Kuwait", label: "(UTC+03:00) Kuwait, Riyadh", offset: 3 },
	{ name: "Asia/Tehran", label: "(UTC+03:30) Tehran", offset: 3.5 },
	{ name: "Asia/Dubai", label: "(UTC+04:00) Abu Dhabi, Muscat", offset: 4 },
	{ name: "Asia/Baku", label: "(UTC+04:00) Baku, Tbilisi, Yerevan", offset: 4 },
	{ name: "Asia/Kabul", label: "(UTC+04:30) Kabul", offset: 4.5 },
	{ name: "Asia/Karachi", label: "(UTC+05:00) Islamabad, Karachi", offset: 5 },
	{ name: "Asia/Kolkata", label: "(UTC+05:30) Mumbai, Kolkata, New Delhi", offset: 5.5 },
	{ name: "Asia/Kathmandu", label: "(UTC+05:45) Kathmandu", offset: 5.75 },
	{ name: "Asia/Dhaka", label: "(UTC+06:00) Dhaka, Astana", offset: 6 },
	{ name: "Asia/Yangon", label: "(UTC+06:30) Yangon (Rangoon)", offset: 6.5 },
	{ name: "Asia/Bangkok", label: "(UTC+07:00) Bangkok, Hanoi, Jakarta", offset: 7 },
	{ name: "Asia/Hong_Kong", label: "(UTC+08:00) Hong Kong, Beijing, Singapore", offset: 8 },
	{ name: "Asia/Shanghai", label: "(UTC+08:00) Shanghai, Taipei", offset: 8 },
	{ name: "Australia/Perth", label: "(UTC+08:00) Perth", offset: 8 },
	{ name: "Asia/Tokyo", label: "(UTC+09:00) Tokyo, Seoul, Osaka", offset: 9 },
	{ name: "Australia/Adelaide", label: "(UTC+09:30) Adelaide", offset: 9.5 },
	{ name: "Australia/Darwin", label: "(UTC+09:30) Darwin", offset: 9.5 },
	{ name: "Australia/Sydney", label: "(UTC+10:00) Sydney, Melbourne, Brisbane", offset: 10 },
	{ name: "Australia/Hobart", label: "(UTC+10:00) Hobart", offset: 10 },
	{ name: "Pacific/Guam", label: "(UTC+10:00) Guam, Port Moresby", offset: 10 },
	{ name: "Pacific/Noumea", label: "(UTC+11:00) Solomon Islands, New Caledonia", offset: 11 },
	{ name: "Pacific/Auckland", label: "(UTC+12:00) Auckland, Wellington", offset: 12 },
	{ name: "Pacific/Fiji", label: "(UTC+12:00) Fiji, Kamchatka", offset: 12 },
	{ name: "Pacific/Tongatapu", label: "(UTC+13:00) Nuku'alofa", offset: 13 }
];

// Populate timezone dropdown and auto-detect user's timezone
function initializeTimezones() {
	const select = document.getElementById('timezone');
	select.innerHTML = '';
	
	// Add all timezones to dropdown
	timezones.forEach(tz => {
		const option = document.createElement('option');
		option.value = tz.name;
		option.textContent = tz.label;
		option.dataset.offset = tz.offset;
		select.appendChild(option);
	});
	
	// Try to detect and select user's timezone
	try {
		const userTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
		const matchingOption = Array.from(select.options).find(opt => opt.value === userTimezone);
		
		if (matchingOption) {
			select.value = userTimezone;
		} else {
			// Fallback: find timezone by offset if exact match not found
			const d = new Date();
			const offsetHours = -d.getTimezoneOffset() / 60;
			const closestTz = timezones.find(tz => Math.abs(tz.offset - offsetHours) < 0.1);
			if (closestTz) {
				select.value = closestTz.name;
			}
		}
	} catch (e) {
		// If detection fails, use first option
		if (select.options.length > 0) {
			select.selectedIndex = 0;
		}
	}
	
	// Update the hidden offset field
	timezoneChanged();
}

// Called when timezone dropdown changes
function timezoneChanged() {
	const select = document.getElementById('timezone');
	const selectedOption = select.options[select.selectedIndex];
	if (selectedOption && selectedOption.dataset.offset) {
		const offset = parseFloat(selectedOption.dataset.offset);
		myzon = offset;
		document.getElementById('zon').value = offset;
		getTimes();
	}
}


// Callback function for Google Maps API
function initMap() {
	setupMap();
}

function setupMap() {
	getLocationFromBrowser();
	showGoogleMaps();
}

function getLocationFromBrowser() {
	// Get geolocation from Browser
	const errorDisplay = document.getElementById("errormessage");
	
	if (!navigator.geolocation) {
		showErrorMessage("Geolocation is not supported by this browser. Using default location.");
		// Still try to get times for default location
		displayTimeZone();
		getTimes();
		return;
	}
	
	// Show loading message
	if (errorDisplay) {
		errorDisplay.textContent = "Getting your location...";
		errorDisplay.style.color = "#666";
	}
	
	navigator.geolocation.getCurrentPosition(
		showgeolocation,
		showError,
		{
			enableHighAccuracy: true,
			timeout: 10000,
			maximumAge: 0
		}
	);
}

function showgeolocation(position) {
	// Display lat and lon
	mylat = parseFloat(position.coords.latitude).toFixed(4);
	mylon = parseFloat(position.coords.longitude).toFixed(4);
	
	updateInputField("lat", mylat);
	updateInputField("lon", mylon);
	moveMarker();
	updateCalLink();
	displayTimeZone();
	getTimes();
}

function updateInputField(fieldId, value) {
	const element = document.getElementById(fieldId);
	if (element) {
		element.value = value;
	}
}

function moveMarker() {
	if (!marker || !myMap) {
		return;
	}
	
	const myLatLon = new google.maps.LatLng(mylat, mylon);
	
	// AdvancedMarkerElement doesn't have setVisible, just update position and pan
	marker.position = myLatLon;
	myMap.panTo(myLatLon);
}

function refresh() {
	getLocationFromBrowser();
}

function displayTimeZone() {
	initializeTimezones();
}

function showGoogleMaps() {
	try {
		const latlon = new google.maps.LatLng(mylat, mylon);
		const mapholder = document.getElementById('mapholder');
		
		if (!mapholder) {
			return;
		}
		
		// Map sizing - now fixed height from CSS
		mapholder.style.width = '100%';
		
		const myOptions = {
			center: latlon,
			zoom: 8,
			mapTypeId: google.maps.MapTypeId.ROADMAP,
			mapTypeControl: true,
			zoomControl: true,
			streetViewControl: false,
			fullscreenControl: true,
			mapId: 'f992b8b594ad429e83d19e2f'
		};
		
		myMap = new google.maps.Map(mapholder, myOptions);
		
		marker = new google.maps.marker.AdvancedMarkerElement({
			position: latlon,
			map: myMap,
			title: "Move marker to your location.",
			gmpDraggable: true
		});
		
		addListener(marker);
	} catch (error) {
		showErrorMessage('Failed to load map. Please refresh the page.');
	}
}

// Detect when the marker is moved or dragged by the user
function addListener(marker) {
	google.maps.event.addListener(marker, 'dragend', (event) => {
		mylat = parseFloat(event.latLng.lat()).toFixed(4);
		mylon = parseFloat(event.latLng.lng()).toFixed(4);
		
		updateInputField("lat", mylat);
		updateInputField("lon", mylon);
		getTimes();
		updateCalLink();
	});
}

// Detect when user changed lat or long input, and move the marker in the map
function SpinnersChanged() {
	const newLat = $('#lat').val();
	const newLon = $('#lon').val();
	
	// Validate coordinates
	if (newLat < -90 || newLat > 90 || newLon < -180 || newLon > 180) {
		showErrorMessage('Invalid coordinates. Latitude: -90 to 90, Longitude: -180 to 180');
		return;
	}
	
	mylat = newLat;
	mylon = newLon;
	
	moveMarker();
	getTimes();
	updateCalLink();
	clearErrorMessage();
}

// Change the calendar link after a lat/lon change
function updateCalLink() {
	const calendarLink = document.getElementById("callink");
	if (calendarLink) {
		calendarLink.href = `calendar?lat=${mylat}&lon=${mylon}&zon=${myzon}`;
	}
}

// Change the values in the spinners after a map change
function updateSpinners() {
	updateInputField("lat", mylat);
	updateInputField("lon", mylon);
	moveMarker();
	updateCalLink();
}

// Get the rise and set times from the server
const getTimes = function () {
	const zon = $('#zon').val();
	myzon = zon;
	
	$.getJSON(`gettimes?lon=${mylon}&lat=${mylat}&zon=${zon}`)
		.done((json) => {
			if (json.Rise === "error" || json.Set === "error") {
				showErrorMessage('Unable to calculate moon times for this location.');
			} else if (json.AlwaysAbove) {
				updateInputField("Rise", "Always above horizon");
				updateInputField("Set", "Always above horizon");
				clearErrorMessage();
			} else if (json.AlwaysBelow) {
				updateInputField("Rise", "Always below horizon");
				updateInputField("Set", "Always below horizon");
				clearErrorMessage();
			} else if (json.Rise && json.Set) {
				updateInputField("Rise", json.Rise);
				updateInputField("Set", json.Set);
				clearErrorMessage();
			}
		})
		.fail((jqXHR, textStatus, errorThrown) => {
			showErrorMessage('Failed to get moon rise/set times. Please try again.');
		});
};

function showError(error) {
	let message = '';
	
	switch (error.code) {
		case error.PERMISSION_DENIED:
			message = "Location access denied. Please enable location permissions or enter coordinates manually.";
			break;
		case error.POSITION_UNAVAILABLE:
			message = "Location information is unavailable. Please enter coordinates manually.";
			break;
		case error.TIMEOUT:
			message = "Location request timed out. Using default location.";
			break;
		case error.UNKNOWN_ERROR:
			message = "An unknown error occurred while getting your location.";
			break;
		default:
			message = "Unable to get your location.";
	}
	
	showErrorMessage(message);
	
	// Still populate timezone and get times for default location
	displayTimeZone();
	getTimes();
}

function showErrorMessage(message) {
	const errorElement = document.getElementById("errormessage");
	if (errorElement) {
		errorElement.textContent = message;
		errorElement.style.color = '#d32f2f';
		errorElement.setAttribute('role', 'alert');
		errorElement.setAttribute('aria-live', 'polite');
	}
}

function clearErrorMessage() {
	const errorElement = document.getElementById("errormessage");
	if (errorElement) {
		errorElement.textContent = '';
		errorElement.removeAttribute('role');
		errorElement.removeAttribute('aria-live');
	}
}

// Handle window resize for responsive map
window.addEventListener('resize', () => {
	if (myMap) {
		google.maps.event.trigger(myMap, 'resize');
	}
});