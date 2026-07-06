// 笔记页锁屏
var ScreenLock = {
	locked: false,
	timer: null,
	timeoutMinutes: 10,
	enabled: true,
	defaultWallpaper: '/images/slider/all.gif',

	init: function() {
		// 确保锁屏层在 body 最顶层，避免被页面 blur/filter 影响
		$('#screenLockMask').appendTo('body');
		this.loadSettings();
		this.renderWallpaper();
		this.bindEvents();
		if (this.enabled) {
			this.resetTimer();
		}
	},

	loadSettings: function() {
		var tm = UserInfo.LockTimeoutMinutes;
		if (!UserInfo.LockConfigured) {
			this.timeoutMinutes = 10;
			this.enabled = true;
		} else if (tm === 0) {
			this.enabled = false;
			this.timeoutMinutes = 0;
		} else {
			this.enabled = true;
			this.timeoutMinutes = tm || 10;
		}
	},

	getWallpaper: function() {
		return UserInfo.LockWallpaper || this.defaultWallpaper;
	},

	renderWallpaper: function() {
		$('#screenLockBg').css('background-image', 'url(' + this.getWallpaper() + ')');
	},

	bindEvents: function() {
		var self = this;
		var activityEvents = 'mousedown mousemove keydown scroll touchstart click';
		$(document).on(activityEvents, function() {
			if (!self.locked && self.enabled) {
				self.resetTimer();
			}
		});

		$('#screenLockUnlockBtn').click(function() {
			self.tryUnlock();
		});
		$('#screenLockPwd').keydown(function(e) {
			if (e.keyCode === 13) {
				self.tryUnlock();
			}
		});
		$('#lockNowBtn').click(function(e) {
			e.preventDefault();
			self.lock();
		});
		$('#screenLockSettingsLink').click(function() {
			window.open('/member/user/lock', '_blank');
		});

		document.addEventListener('visibilitychange', function() {
			if (document.hidden && self.enabled && !self.locked) {
				self.resetTimer();
			}
		});
	},

	resetTimer: function() {
		var self = this;
		if (this.timer) {
			clearTimeout(this.timer);
		}
		if (!this.enabled || this.timeoutMinutes <= 0) {
			return;
		}
		this.timer = setTimeout(function() {
			self.lock();
		}, this.timeoutMinutes * 60 * 1000);
	},

	lock: function() {
		if (this.locked) {
			return;
		}
		this.locked = true;
		if (this.timer) {
			clearTimeout(this.timer);
			this.timer = null;
		}
		this.renderWallpaper();
		this.updateLockPanel();
		$('#screenLockMask').show();
		$('body').addClass('screen-locked');
		setTimeout(function() {
			$('#screenLockPwd').focus();
		}, 100);
	},

	unlock: function() {
		this.locked = false;
		$('#screenLockMask').hide();
		$('#screenLockPwd').val('');
		$('#screenLockMsg').hide();
		$('body').removeClass('screen-locked');
		if (this.enabled) {
			this.resetTimer();
		}
	},

	updateLockPanel: function() {
		var username = UserInfo.Username || '';
		var logo = UserInfo.Logo || '/images/blog/default_avatar.png';
		$('#screenLockAvatar').attr('src', logo);
		$('#screenLockUsername').text(username);
		$('#screenLockTime').text(new Date().toLocaleString());

		if (UserInfo.LockHasPwd) {
			$('#screenLockPwdGroup').show();
			$('#screenLockNoPwdTips').hide();
		} else {
			$('#screenLockPwdGroup').hide();
			$('#screenLockNoPwdTips').show();
		}
	},

	tryUnlock: function() {
		var self = this;
		if (!UserInfo.LockHasPwd) {
			this.unlock();
			return;
		}
		var pwd = $('#screenLockPwd').val();
		if (!pwd) {
			showAlert('#screenLockMsg', getMsg('inputPassword'), 'danger');
			return;
		}
		post('/user/verifyLockPwd', {lockPwd: pwd}, function(e) {
			if (e.Ok) {
				self.unlock();
			} else {
				showAlert('#screenLockMsg', getMsg('lockPasswordError'), 'danger');
			}
		});
	}
};

function initScreenLock() {
	ScreenLock.init();
}
