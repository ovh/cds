import { Injectable } from '@angular/core';

@Injectable()
export class HelpersService {
  constructor() {

  }

  getBrightness(rgb) {
    let result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(rgb);
    return result ?
      0.2126 * parseInt(result[1], 16) +
      0.7152 * parseInt(result[2], 16) +
      0.0722 * parseInt(result[3], 16) : 0;
  }

  getBrightnessColor(rgb) {
    if (this.getBrightness(rgb) > 130) {
      return '#000000';
    }
    return '#ffffff';
  }
}
