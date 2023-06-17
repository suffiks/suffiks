export class ClientError {
  public code: u8;
  constructor(c: usize) {
    this.code = u8(c);
  }

  public isUnknown(): bool {
    return this.code == 0;
  }

  public isNotFound(): bool {
    return this.code == 1;
  }

  public isAlreadyExists(): bool {
    return this.code == 2;
  }

  public isInvalid(): bool {
    return this.code == 3;
  }

  public isForbidden(): bool {
    return this.code == 4;
  }

  public isConflict(): bool {
    return this.code == 5;
  }

  public isBadRequest(): bool {
    return this.code == 6;
  }

  public isGone(): bool {
    return this.code == 7;
  }

  public isInternalError(): bool {
    return this.code == 8;
  }

  public isMethodNotSupported(): bool {
    return this.code == 9;
  }

  public isNotAcceptable(): bool {
    return this.code == 10;
  }

  public isEntityTooLarge(): bool {
    return this.code == 11;
  }

  public isResourceExpired(): bool {
    return this.code == 12;
  }

  public isServerTimeout(): bool {
    return this.code == 13;
  }

  public isServiceUnavailable(): bool {
    return this.code == 14;
  }

  public isTimeout(): bool {
    return this.code == 15;
  }

  public isTooManyRequests(): bool {
    return this.code == 16;
  }

  public isUnauthorized(): bool {
    return this.code == 17;
  }

  public isUnexpectedObject(): bool {
    return this.code == 18;
  }

  public isUnexpectedServerError(): bool {
    return this.code == 19;
  }

  public isUnsupportedMediaType(): bool {
    return this.code == 20;
  }

  public toString(): string {
    switch (this.code) {
      case 0:
        return "Unknown";
      case 1:
        return "NotFound";
      case 2:
        return "AlreadyExists";
      case 3:
        return "Invalid";
      case 4:
        return "Forbidden";
      case 5:
        return "Conflict";
      case 6:
        return "BadRequest";
      case 7:
        return "Gone";
      case 8:
        return "InternalError";
      case 9:
        return "MethodNotSupported";
      case 10:
        return "NotAcceptable";
      case 11:
        return "EntityTooLarge";
      case 12:
        return "ResourceExpired";
      case 13:
        return "ServerTimeout";
      case 14:
        return "ServiceUnavailable";
      case 15:
        return "Timeout";
      case 16:
        return "TooManyRequests";
      case 17:
        return "Unauthorized";
      case 18:
        return "UnexpectedObject";
      case 19:
        return "UnexpectedServerError";
      case 20:
        return "UnsupportedMediaType";
      default:
        return "Unknown";
    }
  }
}
