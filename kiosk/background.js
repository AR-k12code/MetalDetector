function serializeArrayBuffer(buf) {
	let binary = '';
	let bytes = new Uint8Array(buf);
	let len = bytes.byteLength;
	for (var i = 0; i < len; i++) {
		binary += String.fromCharCode(bytes[i]);
	}
	return window.btoa(binary);
};

function deserializeArrayBuffer(str) {
	let binary_string = window.atob(str);
	let len = binary_string.length;
	let bytes = new Uint8Array(len);
	for (var i = 0; i < len; i++) {
		bytes[i] = binary_string.charCodeAt(i);
	}
	return bytes.buffer;
};

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
	chrome.storage.local.set({ sessionStart: timestamp() }, sendReport());
	console.log('session start');
}

chrome.runtime.onStartup.addListener(sessionStart);
chrome.runtime.onInstalled.addListener(sessionStart);

chrome.alarms.onAlarm.addListener(sendReport);

async function sendReport() {
	// get settings
	const settings = await new Promise((resolve, reject) =>
		chrome.storage.managed.get(['server', 'useAuth'], s => resolve(s))
	);
	
	// get report data
	const data = await new Promise((resolve, reject) =>
		reportData(d => resolve(d))
	);
	
	// add challengeResp to data if useAuth is enabled
	if(settings.useAuth) {
		// fetch challenge
		const resp = await fetch(settings.server);
		const respObj = await resp.json();
		data.challengeResp = await new Promise((resolve, reject) =>
			chrome.enterprise.platformKeys.challengeMachineKey(
				deserializeArrayBuffer(respObj.challenge),
				false,
				r => resolve(serializeArrayBuffer(r))
			)
		);
	}
	
	// send data to server
	fetch(settings.server, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(data)
	});

	console.log('send report');
}

function reportData(callback) {
	chrome.enterprise.deviceAttributes.getDeviceSerialNumber(serial =>
		chrome.storage.managed.get({ maxGeoAge: 300 }, settings =>
			chrome.storage.local.get('sessionStart', storage =>
				navigator.geolocation.getCurrentPosition(geo => {
					callback({
						timestamp: timestamp(),
						email: 'KIOSK',
						serial: serial,
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
	);
}

chrome.app.runtime.onLaunched.addListener(() =>
	chrome.storage.managed.get('display', settings =>
		chrome.enterprise.deviceAttributes.getDeviceSerialNumber(serial =>
			chrome.storage.local.set(
				{
					'serial': serial,
					'display': settings.display
				},
				() => chrome.app.window.create('window.html')
			)
		)
	)
);
