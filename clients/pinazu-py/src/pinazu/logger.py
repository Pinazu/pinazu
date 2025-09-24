import logging
from logging import Logger
from pythonjsonlogger.json import JsonFormatter


class CustomFormatter(logging.Formatter):
    """
    Logging Formatter to add colors and align messages based on log level.
    """

    grey = "\x1b[38;20m"
    green = "\x1b[32;20m"
    yellow = "\x1b[33;20m"
    light_yellow = "\x1b[93;20m"
    red = "\x1b[31;20m"
    bold_red = "\x1b[31;1m"
    cyan = "\x1b[36;20m"
    white = "\x1b[37;20m"
    pink = "\x1b[95;20m"
    reset = "\x1b[0m"

    # Define format with specific colors for each component
    # Use '%(levelname)-8s' for left-aligned padding up to 8 characters
    # Longest standard level name is CRITICAL (8 chars)
    log_format = (
        f"{cyan}%(asctime)s{reset} %(name)s {reset}"
        f"%(log_color)s%(levelname)-8s{reset}"  # Padded levelname
        f"%(message)s{reset}"
    )

    LOG_LEVEL_COLORS = {
        logging.DEBUG: grey,
        logging.INFO: green,
        logging.WARNING: yellow,
        logging.ERROR: red,
        logging.CRITICAL: bold_red,
    }

    def format(self, record):
        # Add log_color to the record based on level
        record.log_color = self.LOG_LEVEL_COLORS.get(
            record.levelno,
            self.white,
        )
        formatter = logging.Formatter(
            self.log_format,
            datefmt="%Y-%m-%d %H:%M:%S",
        )
        return formatter.format(record)


class CustomLogger(Logger):
    """Custom logger class with a custom formatter."""

    def __init__(
        self,
        log_level=logging.DEBUG,
        name=__name__,
        use_json=False,
    ):
        super().__init__(name)
        self.setLevel(log_level)  # Set the minimum log level

        # Prevent adding multiple handlers
        # if logger is instantiated multiple times
        if not self.handlers:
            ch = logging.StreamHandler()
            ch.setLevel(log_level)  # Set the minimum log level
            if use_json:
                formatter = JsonFormatter(
                    "%(asctime)s %(levelname)s %(name)s %(message)s %(exc_info)s",  # noqa: E501
                    json_indent=None,
                    rename_fields={
                        "asctime": "timestamp",
                        "levelname": "level",
                        "message": "msg",
                    },
                )
            else:
                formatter = CustomFormatter()
            ch.setFormatter(formatter)
            self.addHandler(ch)
