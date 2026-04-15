export type TextPosition =
  | 'top_left' | 'top_center' | 'top_right'
  | 'mid_left' | 'mid_center' | 'mid_right'
  | 'bottom_left' | 'bottom_center' | 'bottom_right';

export interface TextStyleConfig {
  fontSize: number;
  fontFamily: string;
  textColor: string;
  backgroundColor: string | null;
  padding: number;
  cornerRadius: number;
  borderColor: string | null;
  borderWidth: number;
  shadowColor: string | null;
  shadowOffset: number;
  isBold: boolean;
  isItalic: boolean;
  allCaps: boolean;
  verticalOffset: number;
}

export interface TimedTextOverlay {
  text: string;
  applyToWholeVideo: boolean;
  startTime: number;
  endTime: number | null;
  position: TextPosition;
  style: TextStyleConfig;
}

export interface TextOverlayConfig {
  enabled: boolean;
  overlays: TimedTextOverlay[];
}

export const DEFAULT_STYLE: TextStyleConfig = {
  fontSize: 48,
  fontFamily: 'Arial',
  textColor: '#FFFFFF',
  backgroundColor: null,
  padding: 0,
  cornerRadius: 0,
  borderColor: null,
  borderWidth: 1,
  shadowColor: null,
  shadowOffset: 2,
  isBold: false,
  isItalic: false,
  allCaps: false,
  verticalOffset: 0,
};

export const DEFAULT_OVERLAY: TimedTextOverlay = {
  text: 'Novo Texto',
  applyToWholeVideo: false,
  startTime: 0,
  endTime: null,
  position: 'mid_center',
  style: { ...DEFAULT_STYLE },
};
