// purechat widget
window.purechatApi = {
	l: [],
	t: [],
	on: function() {
		this.l.push(arguments);
	}
};
(function() {
	var done = false;
	var script = document.createElement('script');
	script.async = true;
	script.type = 'text/javascript';
	script.src = 'https://app.purechat.com/VisitorWidget/WidgetScript';
	document.getElementsByTagName('HEAD').item(0).appendChild(script);
	script.onreadystatechange = script.onload = function(e) {
		if (!done && (!this.readyState || this.readyState == 'loaded' || this.readyState == 'complete')) {
			var w = new PCWidget({
				c: '9d8508c5-0690-4528-91ae-07ad27bf4456',
				f: true
			});
			done = true;
		}
	};
})();

// fill in serial number
setTimeout(() =>
	chrome.storage.local.get('serial', storage =>
		document.getElementById('serial').innerText = storage.serial
	),
	5000
);
