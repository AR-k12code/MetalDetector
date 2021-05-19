// TODO: use chrome.enterprise.platformKeys for authentication

function timestamp() {
	return Math.floor(Date.now() / 1000);
}

chrome.alarms.get(alarm => {
	chrome.runtime.getPlatformInfo(platform =>
		chrome.storage.managed.get({ interval: 5 }, settings => {
			if (
				platform.os === 'cros' &&
				(!alarm || alarm.periodInMinutes != settings.interval)
			) {
				chrome.alarms.create({ periodInMinutes: settings.interval });
				console.log('setup alarm');
			}
		})
	);
});

function sessionStart() {
	chrome.storage.local.set({ sessionStart: timestamp() });
	console.log('session start');
}

chrome.runtime.onStartup.addListener(sessionStart);
chrome.runtime.onInstalled.addListener(sessionStart);

chrome.alarms.onAlarm.addListener(() => {
	chrome.storage.managed.get('server', settings =>
		reportData(data =>
			fetch(settings.server, {
				method: 'POST',
				mode: 'no-cors',
				// this is not text/plain, but it lets us skip CORS
				headers: { 'Content-Type': 'text/plain' },
				body: JSON.stringify(data)
			})
		)
	);
	console.log('send report');
});

function reportData(callback) {
	chrome.enterprise.deviceAttributes.getDeviceSerialNumber(serial =>
		chrome.enterprise.networkingAttributes.getNetworkDetails(net =>
			chrome.storage.managed.get({ maxGeoAge: 300 }, settings =>
				chrome.storage.local.get('sessionStart', storage =>
					navigator.geolocation.getCurrentPosition(geo => {
						callback({
							timestamp: timestamp(),
							email: 'KIOSK',
							serial: serial,
							ipv4: net.ipv4,
							ipv6: net.ipv6,
							geoTimestamp: Math.floor(geo.timestamp / 1000),
							latitude: geo.coords.latitude,
							longitude: geo.coords.longitude,
							accuracy: geo.coords.accuracy,
							sessionStart: storage.sessionStart
						});
					}, err => {
						callback({
							timestamp: timestamp(),
							email: 'KIOSK',
							serial: serial,
							ipv4: net.ipv4,
							ipv6: net.ipv6,
							sessionStart: storage.sessionStart,
						});
						console.log('geolocation failed', err);
					}, {
						enableHighAccuracy: true,
						timeout: 10 * 1000,
						maximumAge: settings.maxGeoAge * 1000
					})
				)
			)
		)
	);
}

chrome.app.runtime.onLaunched.addListener(() => {
	// get the device serial, save it, then open a window
	chrome.enterprise.deviceAttributes.getDeviceSerialNumber(serial =>
		chrome.storage.local.set({ "serial": serial }, () => 
			chrome.app.window.create('window.html')
		)
	);
});
