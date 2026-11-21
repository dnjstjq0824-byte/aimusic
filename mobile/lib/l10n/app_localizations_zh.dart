// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Chinese (`zh`).
class AppLocalizationsZh extends AppLocalizations {
  AppLocalizationsZh([String locale = 'zh']) : super(locale);

  @override
  String get appTitle => 'AVIBE';

  @override
  String get tabHome => '首页';

  @override
  String get tabSearch => '搜索';

  @override
  String get tabPlaylists => '歌单';

  @override
  String get tabPlayer => '播放';

  @override
  String get tabStore => '门店';

  @override
  String get sessionExpired => '会话已过期，请重新登录。';

  @override
  String get loginTitle => '登录 AVIBE';

  @override
  String get loginEmail => '邮箱';

  @override
  String get loginPassword => '密码（8位以上）';

  @override
  String get loginTotp => '两步验证码';

  @override
  String get loginTotpHint => '认证应用6位数字或恢复代码';

  @override
  String get loginSubmit => '登录';

  @override
  String get loginSubmitTotp => '验证并登录';

  @override
  String get loginCreateAccount => '创建账号';

  @override
  String get loginHaveAccount => '我已有账号';

  @override
  String get loginNeedAgeConfirm => '需要确认您已年满14岁。';

  @override
  String get loginTotpPrompt => '请输入认证应用中的6位验证码。';

  @override
  String get loginFailed => '认证失败。';

  @override
  String get loginOrSocial => '或使用社交账号继续';

  @override
  String loginSocialFailed(String provider) {
    return '$provider登录失败。';
  }

  @override
  String get loginSocialCancelled => '已取消社交登录。';

  @override
  String get loginSocialReturnFailed => '返回应用失败，请重试。';

  @override
  String get providerGoogle => 'Google';

  @override
  String get providerKakao => 'Kakao';

  @override
  String get providerApple => 'Apple';

  @override
  String get providerNaver => 'Naver';

  @override
  String get providerLine => 'LINE';

  @override
  String get signupTitle => '创建 AVIBE 账号';

  @override
  String get signupDisplayName => '显示名称';

  @override
  String get signupBirthYear => '出生年份（选填）';

  @override
  String get signupBirthYearHint => '例如：1995';

  @override
  String get signupInviteCode => '邀请码（如需）';

  @override
  String get signupAgeConfirm => '我已年满14岁。（必填，未满14岁无法注册）';

  @override
  String get signupSubmit => '注册';

  @override
  String get signupBackToLogin => '返回登录';

  @override
  String get ageGateTitle => '年满14岁确认';

  @override
  String get ageGateDesc => '您首次使用社交账号登录。使用流媒体和支付功能需要进行年龄确认。';

  @override
  String get ageGateConfirm => '我已年满14岁。（必填）';

  @override
  String get ageGateBirthYear => '出生年份（选填）';

  @override
  String get ageGateBirthYearHint => '例如：1995';

  @override
  String get ageGateFailed => '年龄确认失败。';

  @override
  String get ageGateLogout => '退出登录';

  @override
  String get ageGateStart => '确认并开始';

  @override
  String get settingsTitle => '设置';

  @override
  String get settingsLanguage => '语言';

  @override
  String get settingsLanguageSaved => '语言已保存。';

  @override
  String get settingsLanguageFailed => '语言保存失败。';

  @override
  String get settingsServerNote => '登录后将与服务器语言设置同步。';

  @override
  String get settingsLogout => '退出登录';

  @override
  String get settingsLegal => '条款与公告';

  @override
  String get langKo => '한국어';

  @override
  String get langEn => 'English';

  @override
  String get langJa => '日本語';

  @override
  String get langZh => '简体中文';

  @override
  String get langTh => 'ไทย';

  @override
  String get commonBeta => 'Beta';

  @override
  String get homeEmpty => '还没有公开的歌曲。';

  @override
  String get homeFailed => '无法加载动态。';

  @override
  String get searchTitle => '搜索';

  @override
  String get searchHint => '按歌曲标题或AI提供方搜索';

  @override
  String get searchEmpty => '没有搜索结果。';

  @override
  String get searchFailed => '搜索失败。';

  @override
  String get playlistTitle => '播放列表';

  @override
  String get playlistEmpty => '还没有播放列表，点击 + 按钮创建。';

  @override
  String get playlistPublic => '公开';

  @override
  String get playlistPrivate => '私密';

  @override
  String get playlistRename => '重命名';

  @override
  String get playlistDelete => '删除';

  @override
  String get playlistListFailed => '无法加载列表。';

  @override
  String get playlistNew => '新建播放列表';

  @override
  String get playlistEdit => '编辑播放列表';

  @override
  String get playlistNameLabel => '标题';

  @override
  String get playlistCancel => '取消';

  @override
  String get playlistSave => '保存';

  @override
  String get playlistSaveFailed => '保存失败。';

  @override
  String get playlistDeleteFailed => '删除失败。';

  @override
  String get playlistSaveOrder => '保存顺序';

  @override
  String get playlistAddTrack => '添加歌曲';

  @override
  String playlistDetailEmpty(String title) {
    return '\'$title\'中没有歌曲。\n点击 + 按钮添加。';
  }

  @override
  String get playlistPlay => '播放';

  @override
  String get playlistDetailFailed => '无法加载详情。';

  @override
  String get playlistOrderFailed => '顺序保存失败。';

  @override
  String get playlistClose => '关闭';

  @override
  String get playlistAddFailed => '添加失败。';

  @override
  String get playerTitle => '正在播放';

  @override
  String get playerReportBlock => '举报 / 拉黑';

  @override
  String get playerTabWaveform => '波形';

  @override
  String get playerTabLyrics => '歌词';

  @override
  String get playerTabAi => 'AI与许可';

  @override
  String get playerNoTrack => '当前没有播放歌曲。';

  @override
  String playerError(String detail) {
    return '播放器错误：$detail';
  }

  @override
  String get playerReportReasonTitle => '选择举报原因';

  @override
  String get playerBlockArtist => '拉黑该艺人（合作方）并从队列中移除';

  @override
  String get playerViewBlocks => '查看我的拉黑记录';

  @override
  String get playerBlockReasonDefault => '移动应用用户直接拉黑艺人';

  @override
  String get playerArtistBlocked => '已拉黑该艺人，并立即从播放列表中移除。';

  @override
  String get playerBlockFailed => '拉黑艺人失败。';

  @override
  String get playerReportTitle => '举报';

  @override
  String get playerReportDetailLabel => '详细说明（选填）';

  @override
  String get playerReportDetailHint => '请描述侵权内容';

  @override
  String get playerCancel => '取消';

  @override
  String get playerReport => '举报';

  @override
  String playerReportReceived(String status) {
    return '已收到举报（$status）。';
  }

  @override
  String get playerReportFailed => '举报失败。';

  @override
  String get playerWaveformTitle => '音频波形 (Waveform)';

  @override
  String get playerInstrumental => '这是纯音乐（Instrumental）作品。';

  @override
  String playerNoLyrics(String provider) {
    return '暂无歌词。\n该歌曲由 $provider AI 模型生成。';
  }

  @override
  String playerPromptLabel(String prompt) {
    return '生成提示词：$prompt';
  }

  @override
  String get playerTrackFailed => '无法加载歌曲信息。';

  @override
  String get playerAiLabel => '是否为AI生成';

  @override
  String get playerAiYes => 'AI生成音乐';

  @override
  String get playerAiNo => '普通音乐';

  @override
  String get playerProviderLabel => 'AI提供方';

  @override
  String get playerDisclosureLabel => '告知文案';

  @override
  String get playerNoDisclosure => '无单独告知';

  @override
  String get playerBitrateLabel => '比特率';

  @override
  String get playerKeyLabel => '调性';

  @override
  String get playerPlayCountLabel => '播放次数';

  @override
  String get playerStatusLabel => '状态';

  @override
  String get playerAiNotice =>
      '根据AI生成物标识义务，该音乐的提供方与生成情况已公开。商业使用范围遵循各提供方的许可政策与平台服务条款（TOS）。';

  @override
  String get playerViewTerms => '查看服务条款';

  @override
  String get legalTitle => '条款与公告';

  @override
  String get legalAiTitle => 'AI生成物公告';

  @override
  String get legalAiDesc => 'AI音乐标识与许可告知';

  @override
  String get legalBlocksTitle => '我的拉黑记录';

  @override
  String get legalBlocksDesc => '举报记点导致的使用限制';

  @override
  String get legalWithdrawalTitle => '账号注销说明';

  @override
  String get legalTermsHeader => '服务条款';

  @override
  String get legalRequired => '必填';

  @override
  String legalVersion(String version) {
    return '版本 $version';
  }

  @override
  String get legalConsentRequired => ' · 需同意';

  @override
  String get legalConsent => '同意';

  @override
  String get legalConsentDone => '已记录您的同意。';

  @override
  String get legalConsentFailed => '同意处理失败。';

  @override
  String get legalLoadFailed => '无法加载条款。';

  @override
  String get legalAiHeading => 'AI音乐标识说明';

  @override
  String get legalAiP1 =>
      '本平台所有音乐均标明是否为AI生成。可在播放页面的「AI与许可」选项卡中查看提供方（SUNO、UDIO、AIVA等）、告知文案和比特率。';

  @override
  String get legalAiP2 =>
      'AI音乐的商业使用范围遵循各提供方的许可政策与平台服务条款。如怀疑存在未经授权的声音克隆或采样，可通过播放页面的举报菜单反馈。';

  @override
  String get legalViewTos => '查看服务条款（TOS）';

  @override
  String get legalNoBlocks => '暂无拉黑记录。';

  @override
  String legalStrikeItem(String number, String reason) {
    return '记点 $number · $reason';
  }

  @override
  String get legalNoExpiry => '无到期日';

  @override
  String legalExpires(String date) {
    return '到期：$date';
  }

  @override
  String get legalBlocksFailed => '无法加载拉黑记录。';

  @override
  String get legalWithdrawalHeading => '注销前请确认';

  @override
  String get legalWithdrawalBody =>
      '· 播放列表与播放记录将立即停用并删除。\n· 如有进行中的举报或纠纷，将依法保存后销毁。\n· 点击下方[立即注销]按钮，无需联系客服即可在应用内立即注销。';

  @override
  String get legalLogout => '退出登录';

  @override
  String get legalWithdrawNow => '立即注销';

  @override
  String get storeTitle => '门店模式';

  @override
  String get storeRefreshQueue => '刷新队列';

  @override
  String get storeUnpair => '解除配对';

  @override
  String get storeUnpairTitle => '解除配对';

  @override
  String get storeUnpairBody => '解除此平板的门店配对并停止播放。';

  @override
  String get storeCancel => '取消';

  @override
  String get storeUnpairConfirm => '解除';

  @override
  String get storePairGuide => '请输入在合作方控制台门店管理中签发的8位配对码。';

  @override
  String get storePairCodeLabel => '配对码';

  @override
  String get storePairCodeHint => '例如：A1B2C3D4';

  @override
  String get storePair => '配对';

  @override
  String get storePairFailed => '配对失败。';

  @override
  String storeOpenNow(String business) {
    return '$business · 营业中';
  }

  @override
  String storeOpenNowPlaylist(String business, String playlist) {
    return '$business · 营业中 · $playlist';
  }

  @override
  String storeClosed(String business, String reason) {
    return '$business · $reason';
  }

  @override
  String get storeQueueEmpty => '没有可播放的歌曲。';

  @override
  String get storeNoSelection => '未选择歌曲';

  @override
  String get storePrev => '上一首';

  @override
  String get storePause => '暂停';

  @override
  String get storePlay => '播放';

  @override
  String get storeNext => '下一首';

  @override
  String get errNotStreamable => '该歌曲无法播放（已被拉黑或审核中）。正在前往下一首。';

  @override
  String get errNotFound => '未找到请求的内容。';

  @override
  String get errAuth => '需要登录或认证已过期。';

  @override
  String get errInactive => '该账号已停用，请联系客服。';

  @override
  String get errForbidden => '没有权限。';

  @override
  String get errRateLimited => '请求过于频繁，请稍后重试。';

  @override
  String get errEmailTaken => '该邮箱已注册。';

  @override
  String get errTotpRequired => '需要两步验证码。';

  @override
  String get errAgeConfirm => '需要年满14岁确认，请在个人资料中确认年龄。';

  @override
  String get errConsent => '需要同意必填条款，请在条款页面同意。';

  @override
  String get errTotpInvalid => '验证码不正确，请重新输入。';

  @override
  String get errLoginLocked => '登录尝试次数过多，请15分钟后再试。';

  @override
  String get errProviderNotAllowed => '该提供方不在允许名单中，请选择允许的合作提供方。';

  @override
  String errInvalidInput(String detail) {
    return '输入无效：$detail';
  }

  @override
  String get errGeneric => '发生临时错误。';

  @override
  String get errNetwork => '网络连接失败。';

  @override
  String get errRequest => '请求失败。';

  @override
  String get reasonAiPolicy => '违反AI政策';

  @override
  String get reasonVoiceClone => '未经授权的声音克隆';

  @override
  String get reasonSample => '未经授权的采样使用';

  @override
  String get reasonOther => '其他';

  @override
  String get certTitle => '验证证书';

  @override
  String get certNumber => '证书编号';

  @override
  String get certIssuedAt => '签发日期';

  @override
  String get certPlanTier => '套餐';

  @override
  String get certGates => '验证关卡';

  @override
  String get certGatesPassed => '通过';

  @override
  String get certGatesFailed => '未通过';

  @override
  String get certNotFound => '未找到证书。';

  @override
  String get certLoadFailed => '证书加载失败。';

  @override
  String get playerCertificateLabel => '证书';

  @override
  String get completeEmailTitle => '输入邮箱';

  @override
  String get completeEmailSubtitle => '社交账号未提供邮箱，需要补充输入。';

  @override
  String get completeEmailHint => 'you@example.com';

  @override
  String get completeEmailSubmit => '继续';

  @override
  String get completeEmailBackToLogin => '返回登录';

  @override
  String get completeEmailInvalid => '请输入有效的邮箱地址。';

  @override
  String get completeEmailFailed => '提交失败。';

  @override
  String get completeEmailPendingTitle => '验证邮件已发送';

  @override
  String completeEmailPendingBody(String email) {
    return '我们已向 $email 发送了验证邮件。点击邮件中的链接即可完成注册。';
  }
}
