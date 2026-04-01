export interface SpeedConfig {
  enabled: boolean;
  factor: number; // 1.00–1.10
}

export interface VisualConfig {
  cropEnabled: boolean;
  cropPercent: number;  // 1–5
  zoomEnabled: boolean;
  zoomAmount: number;   // 1.01–1.10
  colorGrading: boolean;
  brightness: number;   // -0.1–0.1
  saturation: number;   // 0.8–1.2
  contrast: number;     // 0.8–1.2
}

export interface CaptionStyleConfig {
  style: 'BOLD' | 'HIGHLIGHT' | 'SUBTITLE';
  fontSize: number;     // 36–72
  textColor: string;    // hex e.g. "#FFFFFF"
  borderWidth: number;  // 0–8
  isBold: boolean;
}

export interface BrandingConfig {
  logoEnabled: boolean;
  logoPath: string;
  logoPosition: 'TOP_LEFT' | 'TOP_RIGHT' | 'BOTTOM_LEFT' | 'BOTTOM_RIGHT';
  logoOpacity: number;  // 0.1–1.0
}

export interface ModeConfig {
  mode: 'ai' | 'longform';
  // AI-specific
  sensitivityPct?: number;    // 50–90
  skipMusicMinutes?: number;  // 0 | 1 | 2 | 3 | 5
  forceRegen?: boolean;
  skipTranscription?: boolean;
  // Duration
  minDurationSecs: number;
  maxDurationSecs: number;
  // Anti-duplication
  antiDuplicate: boolean;
  antiDuplicateMode: 'subtle' | 'aggressive';
  speed: SpeedConfig;
  visual: VisualConfig;
  // Captions
  withCaptions: boolean;
  captionStyle: CaptionStyleConfig;
  // Overlays / Branding
  withOverlays: boolean;
  branding: BrandingConfig;
  // Upload
  autoUpload: boolean;
  privacy: 'public' | 'unlisted' | 'private';
  channelId?: number;
}
